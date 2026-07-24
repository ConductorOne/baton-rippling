package connector

import (
	"context"
	"testing"

	"github.com/conductorone/baton-rippling/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/session"
	"github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/conductorone/baton-sdk/pkg/types/sessions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSessionStore is a minimal in-process sessions.SessionStore backed by a
// map, namespacing keys by the SessionStoreOption-supplied prefix the same
// way the real backends do (bag.Prefix + "/" + key). Only Get/Set are
// exercised by userBuilder.Grants()'s session.GetJSON/SetJSON path; the rest
// are stubbed since this test never calls them.
type fakeSessionStore struct {
	data map[string][]byte
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{data: make(map[string][]byte)}
}

func (f *fakeSessionStore) applyOpts(opt ...sessions.SessionStoreOption) (*sessions.SessionStoreBag, error) {
	bag := &sessions.SessionStoreBag{}
	for _, o := range opt {
		if err := o(context.Background(), bag); err != nil {
			return nil, err
		}
	}
	return bag, nil
}

func (f *fakeSessionStore) namespacedKey(key string, opt ...sessions.SessionStoreOption) (string, error) {
	bag, err := f.applyOpts(opt...)
	if err != nil {
		return "", err
	}
	return bag.Prefix + "/" + key, nil
}

func (f *fakeSessionStore) Get(ctx context.Context, key string, opt ...sessions.SessionStoreOption) ([]byte, bool, error) {
	nk, err := f.namespacedKey(key, opt...)
	if err != nil {
		return nil, false, err
	}
	v, ok := f.data[nk]
	return v, ok, nil
}

func (f *fakeSessionStore) GetMany(ctx context.Context, keys []string, opt ...sessions.SessionStoreOption) (map[string][]byte, []string, error) {
	result := make(map[string][]byte)
	var missing []string
	for _, key := range keys {
		nk, err := f.namespacedKey(key, opt...)
		if err != nil {
			return nil, nil, err
		}
		if v, ok := f.data[nk]; ok {
			result[key] = v
		} else {
			missing = append(missing, key)
		}
	}
	return result, missing, nil
}

func (f *fakeSessionStore) Set(ctx context.Context, key string, value []byte, opt ...sessions.SessionStoreOption) error {
	nk, err := f.namespacedKey(key, opt...)
	if err != nil {
		return err
	}
	f.data[nk] = value
	return nil
}

func (f *fakeSessionStore) SetMany(ctx context.Context, values map[string][]byte, opt ...sessions.SessionStoreOption) error {
	for key, value := range values {
		if err := f.Set(ctx, key, value, opt...); err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeSessionStore) Delete(ctx context.Context, key string, opt ...sessions.SessionStoreOption) error {
	nk, err := f.namespacedKey(key, opt...)
	if err != nil {
		return err
	}
	delete(f.data, nk)
	return nil
}

func (f *fakeSessionStore) Clear(ctx context.Context, opt ...sessions.SessionStoreOption) error {
	f.data = make(map[string][]byte)
	return nil
}

func (f *fakeSessionStore) GetAll(ctx context.Context, pageToken string, opt ...sessions.SessionStoreOption) (map[string][]byte, string, error) {
	return f.data, "", nil
}

func resourceForUserID(t *testing.T, userID string) *v2.Resource {
	t.Helper()
	id, err := resource.NewResourceID(userResourceType, userID)
	require.NoError(t, err)
	return &v2.Resource{Id: id}
}

// TestUserBuilder_ResourceType_SyncTeamsTrue_SkipsEntitlementsOnly proves that when
// the team resource type is being synced, the user resource type advertises only
// the base SkipEntitlements annotation (unchanged from userResourceType) — meaning
// the SDK will still call Grants() so the cross-type team-membership emission runs.
func TestUserBuilder_ResourceType_SyncTeamsTrue_SkipsEntitlementsOnly(t *testing.T) {
	o := newUserBuilder(nil, false, nil, true)

	rt := o.ResourceType(context.Background())
	require.NotNil(t, rt)

	annos := annotations.Annotations(rt.GetAnnotations())
	assert.True(t, annos.Contains(&v2.SkipEntitlements{}))
	assert.False(t, annos.Contains(&v2.SkipEntitlementsAndGrants{}))

	// Returned unchanged, so it must be the exact shared package-level var.
	assert.Same(t, userResourceType, rt)
}

// TestUserBuilder_ResourceType_SyncTeamsFalse_SkipsEntitlementsAndGrants proves that
// when the team resource type is NOT being synced, the user resource type advertises
// SkipEntitlementsAndGrants — since userBuilder.Grants() only ever produces cross-type
// team-membership grants and nothing of its own, this tells the SDK to skip calling
// Entitlements()/Grants() for user resources entirely rather than wastefully invoking
// Grants() only to compute grants for a type that isn't being synced.
func TestUserBuilder_ResourceType_SyncTeamsFalse_SkipsEntitlementsAndGrants(t *testing.T) {
	o := newUserBuilder(nil, false, nil, false)

	rt := o.ResourceType(context.Background())
	require.NotNil(t, rt)

	annos := annotations.Annotations(rt.GetAnnotations())
	assert.True(t, annos.Contains(&v2.SkipEntitlementsAndGrants{}))

	// Must be a clone, never a mutation of the shared package-level var: the
	// base var's own annotations must remain untouched (no SkipEntitlementsAndGrants).
	assert.NotSame(t, userResourceType, rt)
	baseAnnos := annotations.Annotations(userResourceType.GetAnnotations())
	assert.False(t, baseAnnos.Contains(&v2.SkipEntitlementsAndGrants{}))
	assert.True(t, baseAnnos.Contains(&v2.SkipEntitlements{}))
}

// TestUserBuilder_Grants_SyncTeamsTrue_EmitsTeamGrants proves the pre-existing
// behavior is unchanged when syncTeams is true: a worker's cached TeamsID entries
// each produce a team-membership grant.
func TestUserBuilder_Grants_SyncTeamsTrue_EmitsTeamGrants(t *testing.T) {
	ctx := context.Background()
	store := newFakeSessionStore()

	worker := client.Worker{
		UserID:  "user-42",
		TeamsID: []string{"team-1", "team-2"},
		Status:  "ACTIVE",
	}
	require.NoError(t, session.SetJSON(ctx, store, "user-42", worker, sessions.WithPrefix(workersSessionPrefix)))

	o := newUserBuilder(nil, false, nil, true)

	grants, _, err := o.Grants(ctx, resourceForUserID(t, "user-42"), resource.SyncOpAttrs{Session: store})
	require.NoError(t, err)
	require.Len(t, grants, 2)

	gotTeamIDs := make([]string, 0, len(grants))
	for _, g := range grants {
		teamID := g.GetEntitlement().GetResource().GetId()
		assert.Equal(t, teamResourceType.Id, teamID.GetResourceType())
		assert.Equal(t, entitlement.NewEntitlementID(g.GetEntitlement().GetResource(), teamMembership), g.GetEntitlement().GetId())
		gotTeamIDs = append(gotTeamIDs, teamID.GetResource())
	}
	assert.ElementsMatch(t, []string{"team-1", "team-2"}, gotTeamIDs)
}
