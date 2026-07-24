package connector

import (
	"context"
	"testing"

	"github.com/conductorone/baton-rippling/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
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

// TestUserBuilder_Grants_SyncTeamsFalse_NoPanic proves that Grants() short-circuits
// before touching opts.Session at all when syncTeams is false: opts is a zero-value
// SyncOpAttrs{}, so opts.Session is a nil interface. If the gate weren't the very
// first thing in the function body, the session.GetJSON call below it would
// dereference the nil interface and panic.
func TestUserBuilder_Grants_SyncTeamsFalse_NoPanic(t *testing.T) {
	o := newUserBuilder(nil, false, nil, false)

	var grants []*v2.Grant
	var results *resource.SyncOpResults
	var err error
	assert.NotPanics(t, func() {
		grants, results, err = o.Grants(context.Background(), resourceForUserID(t, "user-42"), resource.SyncOpAttrs{})
	})

	assert.NoError(t, err)
	assert.Nil(t, grants)
	assert.Nil(t, results)
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
