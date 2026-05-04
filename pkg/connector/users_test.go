package connector

import (
	"testing"

	"github.com/conductorone/baton-rippling/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/stretchr/testify/assert"
)

func TestDeriveUserStatus(t *testing.T) {
	cases := []struct {
		name       string
		worker     *client.Worker
		wantStatus v2.UserTrait_Status_Status
		wantReason string
	}{
		{
			name:       "nil worker — cache miss or admin/system account",
			worker:     nil,
			wantStatus: v2.UserTrait_Status_STATUS_UNSPECIFIED,
			wantReason: reasonWorkerNil,
		},
		{
			name:       "ACTIVE — happy path",
			worker:     &client.Worker{Status: "ACTIVE"},
			wantStatus: v2.UserTrait_Status_STATUS_ENABLED,
			wantReason: "",
		},
		{
			name:       "TERMINATED",
			worker:     &client.Worker{Status: "TERMINATED"},
			wantStatus: v2.UserTrait_Status_STATUS_DISABLED,
			wantReason: reasonWorkerTerminated,
		},
		{
			name:       "HIRED — pre-hire (PR 1: still UNSPECIFIED)",
			worker:     &client.Worker{Status: "HIRED"},
			wantStatus: v2.UserTrait_Status_STATUS_UNSPECIFIED,
			wantReason: reasonWorkerPreHire,
		},
		{
			name:       "ACCEPTED — pre-hire (PR 1: still UNSPECIFIED)",
			worker:     &client.Worker{Status: "ACCEPTED"},
			wantStatus: v2.UserTrait_Status_STATUS_UNSPECIFIED,
			wantReason: reasonWorkerPreHire,
		},
		{
			name:       "INIT — draft worker",
			worker:     &client.Worker{Status: "INIT"},
			wantStatus: v2.UserTrait_Status_STATUS_UNSPECIFIED,
			wantReason: reasonWorkerInit,
		},
		{
			name:       "empty status — malformed/partial Rippling response",
			worker:     &client.Worker{Status: ""},
			wantStatus: v2.UserTrait_Status_STATUS_UNSPECIFIED,
			wantReason: reasonWorkerStatusEmpty,
		},
		{
			name:       "LEAVE — undocumented future status",
			worker:     &client.Worker{Status: "LEAVE"},
			wantStatus: v2.UserTrait_Status_STATUS_UNSPECIFIED,
			wantReason: reasonWorkerStatusUnknown,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotStatus, gotReason := deriveUserStatus(tc.worker)
			assert.Equal(t, tc.wantStatus, gotStatus)
			assert.Equal(t, tc.wantReason, gotReason)
		})
	}
}

func TestUserResource_NoWorker(t *testing.T) {
	user := client.User{
		ID:          "user-1",
		Username:    "alice",
		Active:      true,
		Locale:      "en-US",
		CreatedAt:   "2024-01-01T00:00:00Z",
		DisplayName: "Alice Smith",
		Name: client.Name{
			GivenName:           "Alice",
			FamilyName:          "Smith",
			Formatted:           "Alice Smith",
			PreferredGivenName:  "Ali",
			PreferredFamilyName: "Smith",
		},
		Emails: []client.Email{{Value: "alice@example.com"}},
	}

	r, err := userResource(user, nil, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Alice Smith", r.DisplayName)
	assert.Equal(t, "user-1", r.Id.Resource)

	profile := r.GetAnnotations()
	assert.NotNil(t, profile)
}

func TestUserResource_WithFullWorker(t *testing.T) {
	user := client.User{
		ID:          "user-2",
		Username:    "bob",
		Active:      true,
		Locale:      "en-US",
		CreatedAt:   "2024-06-15T12:00:00Z",
		DisplayName: "Bob Jones",
		Name: client.Name{
			GivenName:  "Bob",
			FamilyName: "Jones",
			MiddleName: "Michael",
		},
		Emails: []client.Email{{Value: "bob@example.com"}},
		Addresses: []client.Address{
			{
				Type:          "WORK",
				Locality:      "San Francisco",
				Region:        "CA",
				Country:       "US",
				StreetAddress: "123 Main St",
				PostalCode:    "94105",
			},
			{
				Type:     "HOME",
				Locality: "Oakland",
				Region:   "CA",
				Country:  "US",
			},
		},
	}
	worker := &client.Worker{
		ID:           "worker-2",
		UserID:       "user-2",
		Title:        "Senior Engineer",
		Status:       "ACTIVE",
		StartDate:    "2023-01-15",
		EndDate:      "",
		WorkEmail:    "bob@company.com",
		Country:      "US",
		IsManager:    true,
		ManagerID:    "worker-1",
		DepartmentID: "dept-eng-1",
		EmploymentType: &client.EmploymentType{
			Label: "Full-time",
			Name:  "FULL_TIME",
			Type:  "employee",
		},
		Department: &client.Department{
			Name: "Engineering",
		},
		Level: &client.Level{
			Name: "Senior",
		},
		Location: &client.Location{
			WorkLocationID: "loc-sf",
		},
	}

	r, err := userResource(user, worker, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Bob Jones", r.DisplayName)
	assert.Equal(t, "user-2", r.Id.Resource)
}

func TestUserResource_WorkerWithNilNestedStructs(t *testing.T) {
	user := client.User{
		ID:          "user-3",
		Username:    "carol",
		Active:      true,
		Locale:      "en-US",
		CreatedAt:   "2024-03-01T00:00:00Z",
		DisplayName: "Carol Lee",
		Emails:      []client.Email{{Value: "carol@example.com"}},
	}
	worker := &client.Worker{
		ID:             "worker-3",
		UserID:         "user-3",
		Title:          "Product Manager",
		Status:         "ACTIVE",
		IsManager:      false,
		EmploymentType: nil,
		Department:     nil,
		Level:          nil,
		Location:       nil,
	}

	r, err := userResource(user, worker, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Carol Lee", r.DisplayName)
}

func TestUserResource_WorkerEmptyStringsNotIncluded(t *testing.T) {
	user := client.User{
		ID:          "user-4",
		Username:    "dave",
		Active:      false,
		Locale:      "en-US",
		CreatedAt:   "2024-02-01T00:00:00Z",
		DisplayName: "Dave Kim",
		Emails:      []client.Email{{Value: "dave@example.com"}},
	}
	worker := &client.Worker{
		ID:     "worker-4",
		UserID: "user-4",
		Title:  "",
		Status: "TERMINATED",
		EmploymentType: &client.EmploymentType{
			Label: "",
			Name:  "",
			Type:  "",
		},
		Department: &client.Department{Name: ""},
		Level:      &client.Level{Name: ""},
		Location:   &client.Location{WorkLocationID: ""},
	}

	r, err := userResource(user, worker, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Dave Kim", r.DisplayName)
}

func TestUserResource_NoEmails(t *testing.T) {
	user := client.User{
		ID:          "user-5",
		Username:    "eve",
		Active:      true,
		Locale:      "en-US",
		CreatedAt:   "2024-01-01T00:00:00Z",
		DisplayName: "Eve Wu",
		Emails:      nil,
	}

	r, err := userResource(user, nil, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Eve Wu", r.DisplayName)
}

func TestUserResource_ProfileNameAndAddressFields(t *testing.T) {
	user := client.User{
		ID:          "user-7",
		Username:    "grace",
		Active:      true,
		Locale:      "en-US",
		CreatedAt:   "2024-01-01T00:00:00Z",
		DisplayName: "Grace Hopper",
		Name: client.Name{
			GivenName:           "Grace",
			FamilyName:          "Hopper",
			MiddleName:          "Brewster",
			Formatted:           "Grace Brewster Hopper",
			PreferredGivenName:  "Grace",
			PreferredFamilyName: "Hopper",
		},
		Emails: []client.Email{{Value: "grace@example.com"}},
		Addresses: []client.Address{
			{
				Type:          "WORK",
				Locality:      "Arlington",
				Region:        "VA",
				Country:       "US",
				StreetAddress: "1400 Defense Pentagon",
				PostalCode:    "22202",
			},
			{
				Type:     "HOME",
				Locality: "New York",
				Region:   "NY",
				Country:  "US",
			},
			// Second WORK address should be ignored (keep first)
			{
				Type:     "WORK",
				Locality: "Washington",
				Region:   "DC",
				Country:  "US",
			},
		},
	}
	worker := &client.Worker{
		ID:           "worker-7",
		UserID:       "user-7",
		DepartmentID: "dept-42",
		Status:       "ACTIVE",
		Department:   &client.Department{Name: "R&D"},
	}

	r, err := userResource(user, worker, nil, nil)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, "Grace Hopper", r.DisplayName)
	assert.Equal(t, "user-7", r.Id.Resource)

	// Extract profile and verify only the first WORK address is used
	trait, err := resource.GetUserTrait(r)
	if !assert.NoError(t, err) {
		return
	}

	profileFields := trait.GetProfile().GetFields()
	addrVal := profileFields["address"].GetStructValue()
	if assert.NotNil(t, addrVal, "expected address in profile") {
		// First WORK address should be kept
		assert.Equal(t, "Arlington", addrVal.GetFields()["locality"].GetStringValue())
		assert.Equal(t, "VA", addrVal.GetFields()["region"].GetStringValue())
		assert.Equal(t, "US", addrVal.GetFields()["country"].GetStringValue())

		// Second WORK address (Washington, DC) should NOT appear — first one wins
		assert.NotEqual(t, "Washington", addrVal.GetFields()["locality"].GetStringValue())
		assert.NotEqual(t, "DC", addrVal.GetFields()["region"].GetStringValue())
	}
}

func TestUserResource_AddressWithNonWorkType(t *testing.T) {
	user := client.User{
		ID:          "user-8",
		Username:    "hank",
		Active:      true,
		Locale:      "en-US",
		CreatedAt:   "2024-01-01T00:00:00Z",
		DisplayName: "Hank Hill",
		Emails:      []client.Email{{Value: "hank@example.com"}},
		Addresses: []client.Address{
			{
				Type:     "HOME",
				Locality: "Arlen",
				Region:   "TX",
				Country:  "US",
			},
		},
	}

	r, err := userResource(user, nil, nil, nil)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, "Hank Hill", r.DisplayName)

	// Non-WORK addresses should not produce an address field
	trait, err := resource.GetUserTrait(r)
	if !assert.NoError(t, err) {
		return
	}
	assert.Nil(t, trait.GetProfile().GetFields()["address"], "expected no address for non-WORK type")
}

func TestUserResource_AddressAllFieldsEmpty(t *testing.T) {
	user := client.User{
		ID:          "user-9",
		Username:    "irene",
		Active:      true,
		Locale:      "en-US",
		CreatedAt:   "2024-01-01T00:00:00Z",
		DisplayName: "Irene Adler",
		Emails:      []client.Email{{Value: "irene@example.com"}},
		Addresses: []client.Address{
			{
				Type: "WORK",
				// All address fields empty
			},
		},
	}

	r, err := userResource(user, nil, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Irene Adler", r.DisplayName)
}

func TestUserResource_WithWorkLocation(t *testing.T) {
	user := client.User{
		ID:          "user-10",
		Username:    "kate",
		Active:      true,
		Locale:      "en-US",
		CreatedAt:   "2024-01-01T00:00:00Z",
		DisplayName: "Kate Walsh",
		Emails:      []client.Email{{Value: "kate@example.com"}},
	}
	worker := &client.Worker{
		ID:     "worker-10",
		UserID: "user-10",
		Status: "ACTIVE",
		Location: &client.Location{
			WorkLocationID: "loc-nyc",
		},
	}
	workLocation := &client.WorkLocation{
		ID:   "loc-nyc",
		Name: "New York Office",
		Address: &client.Address{
			StreetAddress: "350 Fifth Ave",
			Locality:      "New York",
			Region:        "NY",
			PostalCode:    "10118",
			Country:       "US",
		},
	}

	r, err := userResource(user, worker, workLocation, nil)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, "Kate Walsh", r.DisplayName)

	trait, err := resource.GetUserTrait(r)
	if !assert.NoError(t, err) {
		return
	}
	fields := trait.GetProfile().GetFields()

	assert.Equal(t, "New York Office", fields["work_location_name"].GetStringValue())

	addrVal := fields["work_location_address"].GetStructValue()
	if assert.NotNil(t, addrVal, "expected work_location_address in profile") {
		addrFields := addrVal.GetFields()
		assert.Equal(t, "350 Fifth Ave", addrFields["street_address"].GetStringValue())
		assert.Equal(t, "New York", addrFields["locality"].GetStringValue())
		assert.Equal(t, "NY", addrFields["region"].GetStringValue())
		assert.Equal(t, "10118", addrFields["postal_code"].GetStringValue())
		assert.Equal(t, "US", addrFields["country"].GetStringValue())
	}
}

func TestUserResource_NilWorkLocation(t *testing.T) {
	user := client.User{
		ID:          "user-11",
		Username:    "leo",
		Active:      true,
		Locale:      "en-US",
		CreatedAt:   "2024-01-01T00:00:00Z",
		DisplayName: "Leo Park",
		Emails:      []client.Email{{Value: "leo@example.com"}},
	}
	worker := &client.Worker{
		ID:     "worker-11",
		UserID: "user-11",
		Status: "ACTIVE",
	}

	r, err := userResource(user, worker, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Leo Park", r.DisplayName)
}

func TestUserResource_WorkLocationNameOnly(t *testing.T) {
	user := client.User{
		ID:          "user-12",
		Username:    "maya",
		Active:      true,
		Locale:      "en-US",
		CreatedAt:   "2024-01-01T00:00:00Z",
		DisplayName: "Maya Chen",
		Emails:      []client.Email{{Value: "maya@example.com"}},
	}
	worker := &client.Worker{
		ID:     "worker-12",
		UserID: "user-12",
		Status: "ACTIVE",
	}
	workLocation := &client.WorkLocation{
		ID:   "loc-remote",
		Name: "Remote",
	}

	r, err := userResource(user, worker, workLocation, nil)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, "Maya Chen", r.DisplayName)

	trait, err := resource.GetUserTrait(r)
	if !assert.NoError(t, err) {
		return
	}
	fields := trait.GetProfile().GetFields()

	assert.Equal(t, "Remote", fields["work_location_name"].GetStringValue())
	assert.Nil(t, fields["work_location_address"], "expected no work_location_address when address is nil")
}

func TestUserResource_InvalidCreatedAt(t *testing.T) {
	user := client.User{
		ID:          "user-6",
		Username:    "frank",
		Active:      true,
		CreatedAt:   "not-a-date",
		DisplayName: "Frank",
	}

	_, err := userResource(user, nil, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse created_at")
}

func TestUserResource_CustomFieldsAddedToProfile(t *testing.T) {
	user := client.User{
		ID:          "user-cf-1",
		Username:    "alice",
		Active:      true,
		Locale:      "en-US",
		CreatedAt:   "2024-01-01T00:00:00Z",
		DisplayName: "Alice Smith",
		Emails:      []client.Email{{Value: "alice@example.com"}},
	}
	worker := &client.Worker{
		ID:     "worker-cf-1",
		UserID: "user-cf-1",
		Status: "ACTIVE",
		CustomFields: []client.CustomField{
			{Name: "Scrum Teams", Type: "text", Value: "Digital Onboarding"},
			{Name: "Scrum Team Code", Type: "text", Value: "DOB"},
			{Name: "Other Field", Type: "text", Value: "should be excluded"},
		},
	}

	r, err := userResource(user, worker, nil, []string{"Scrum Teams", "Scrum Team Code"})
	if !assert.NoError(t, err) {
		return
	}

	trait, err := resource.GetUserTrait(r)
	if !assert.NoError(t, err) {
		return
	}
	fields := trait.GetProfile().GetFields()

	assert.Equal(t, "Digital Onboarding", fields["scrum_teams"].GetStringValue())
	assert.Equal(t, "DOB", fields["scrum_team_code"].GetStringValue())
	assert.Nil(t, fields["other_field"], "non-configured custom field should not appear")
}

func TestUserResource_CustomFieldsCaseInsensitive(t *testing.T) {
	user := client.User{
		ID:          "user-cf-2",
		Username:    "bob",
		Active:      true,
		Locale:      "en-US",
		CreatedAt:   "2024-01-01T00:00:00Z",
		DisplayName: "Bob Jones",
		Emails:      []client.Email{{Value: "bob@example.com"}},
	}
	worker := &client.Worker{
		ID:     "worker-cf-2",
		UserID: "user-cf-2",
		Status: "ACTIVE",
		CustomFields: []client.CustomField{
			{Name: "SCRUM TEAMS", Type: "text", Value: "Platform"},
		},
	}

	r, err := userResource(user, worker, nil, []string{"scrum teams"})
	if !assert.NoError(t, err) {
		return
	}

	trait, err := resource.GetUserTrait(r)
	if !assert.NoError(t, err) {
		return
	}
	fields := trait.GetProfile().GetFields()

	assert.Equal(t, "Platform", fields["scrum_teams"].GetStringValue())
}

func TestUserResource_NoCustomFieldsConfig(t *testing.T) {
	user := client.User{
		ID:          "user-cf-3",
		Username:    "carol",
		Active:      true,
		Locale:      "en-US",
		CreatedAt:   "2024-01-01T00:00:00Z",
		DisplayName: "Carol Lee",
		Emails:      []client.Email{{Value: "carol@example.com"}},
	}
	worker := &client.Worker{
		ID:     "worker-cf-3",
		UserID: "user-cf-3",
		Status: "ACTIVE",
		CustomFields: []client.CustomField{
			{Name: "Scrum Teams", Type: "text", Value: "Digital Onboarding"},
		},
	}

	r, err := userResource(user, worker, nil, nil)
	if !assert.NoError(t, err) {
		return
	}

	trait, err := resource.GetUserTrait(r)
	if !assert.NoError(t, err) {
		return
	}
	fields := trait.GetProfile().GetFields()

	assert.Nil(t, fields["scrum_teams"], "custom field should not appear when config is empty")
}

func TestUserResource_CustomFieldsDoNotOverwriteBuiltIn(t *testing.T) {
	user := client.User{
		ID:          "user-cf-4",
		Username:    "dave",
		Active:      true,
		Locale:      "en-US",
		CreatedAt:   "2024-01-01T00:00:00Z",
		DisplayName: "Dave Kim",
		Emails:      []client.Email{{Value: "dave@example.com"}},
	}
	worker := &client.Worker{
		ID:     "worker-cf-4",
		UserID: "user-cf-4",
		Status: "ACTIVE",
		Title:  "Senior Engineer",
		CustomFields: []client.CustomField{
			{Name: "Title", Type: "text", Value: "Should Not Overwrite"},
		},
	}

	r, err := userResource(user, worker, nil, []string{"Title"})
	if !assert.NoError(t, err) {
		return
	}

	trait, err := resource.GetUserTrait(r)
	if !assert.NoError(t, err) {
		return
	}
	fields := trait.GetProfile().GetFields()

	// Built-in "title" from worker.Title should be preserved, not overwritten
	assert.Equal(t, "Senior Engineer", fields["title"].GetStringValue())
}

func TestUserResource_LoginAlias_FromWorkerWorkEmail(t *testing.T) {
	user := client.User{
		ID:          "user-alias-1",
		Username:    "alice.personal@gmail.com",
		Active:      true,
		Locale:      "en-US",
		CreatedAt:   "2024-01-01T00:00:00Z",
		DisplayName: "Alice Smith",
	}
	worker := &client.Worker{
		ID:        "worker-alias-1",
		UserID:    "user-alias-1",
		Status:    "ACTIVE",
		WorkEmail: "alice@company.com",
	}

	r, err := userResource(user, worker, nil, nil)
	if !assert.NoError(t, err) {
		return
	}

	trait, err := resource.GetUserTrait(r)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "alice.personal@gmail.com", trait.GetLogin(), "primary login should remain user.Username")
	assert.Equal(t, []string{"alice@company.com"}, trait.GetLoginAliases(), "worker.WorkEmail should be a login alias")
}

func TestUserResource_LoginAlias_FallbackToWorkTypeEmail(t *testing.T) {
	user := client.User{
		ID:          "user-alias-2",
		Username:    "bob.personal@gmail.com",
		Active:      true,
		CreatedAt:   "2024-01-01T00:00:00Z",
		DisplayName: "Bob Jones",
		Emails: []client.Email{
			{Value: "bob.personal@gmail.com", Type: "HOME"},
			{Value: "bob@company.com", Type: "WORK"},
		},
	}
	// Worker present but WorkEmail empty — the WORK-typed email on user should be used as the alias.
	worker := &client.Worker{
		ID:     "worker-alias-2",
		UserID: "user-alias-2",
		Status: "ACTIVE",
	}

	r, err := userResource(user, worker, nil, nil)
	if !assert.NoError(t, err) {
		return
	}

	trait, err := resource.GetUserTrait(r)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "bob.personal@gmail.com", trait.GetLogin())
	assert.Equal(t, []string{"bob@company.com"}, trait.GetLoginAliases())
}

func TestUserResource_LoginAlias_NoAliasWhenWorkEmailMatchesUsername(t *testing.T) {
	user := client.User{
		ID:          "user-alias-3",
		Username:    "carol@company.com",
		Active:      true,
		CreatedAt:   "2024-01-01T00:00:00Z",
		DisplayName: "Carol Lee",
	}
	worker := &client.Worker{
		ID:        "worker-alias-3",
		UserID:    "user-alias-3",
		Status:    "ACTIVE",
		WorkEmail: "CAROL@company.com",
	}

	r, err := userResource(user, worker, nil, nil)
	if !assert.NoError(t, err) {
		return
	}

	trait, err := resource.GetUserTrait(r)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "carol@company.com", trait.GetLogin())
	assert.Empty(t, trait.GetLoginAliases(), "no alias expected when work email matches username (case-insensitive)")
}

func TestUserResource_LoginAlias_NoAliasWhenNoWorkEmailAvailable(t *testing.T) {
	user := client.User{
		ID:          "user-alias-4",
		Username:    "dave",
		Active:      true,
		CreatedAt:   "2024-01-01T00:00:00Z",
		DisplayName: "Dave Kim",
		Emails: []client.Email{
			{Value: "dave@home.example", Type: "HOME"},
		},
	}

	r, err := userResource(user, nil, nil, nil)
	if !assert.NoError(t, err) {
		return
	}

	trait, err := resource.GetUserTrait(r)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "dave", trait.GetLogin())
	assert.Empty(t, trait.GetLoginAliases())
}
