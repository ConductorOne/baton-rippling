package connector

import (
	"testing"

	"github.com/conductorone/baton-rippling/pkg/client"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/stretchr/testify/assert"
)

func TestUserResource_NoWorker(t *testing.T) {
	user := client.User{
		ID:        "user-1",
		Username:  "alice",
		Active:    true,
		Locale:    "en-US",
		CreatedAt: "2024-01-01T00:00:00Z",
		Name: client.Name{
			DisplayName:         "Alice Smith",
			GivenName:           "Alice",
			FamilyName:          "Smith",
			Formatted:           "Alice Smith",
			PreferredGivenName:  "Ali",
			PreferredFamilyName: "Smith",
		},
		Emails: []client.Email{{Value: "alice@example.com"}},
	}

	r, err := userResource(user, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Alice Smith", r.DisplayName)
	assert.Equal(t, "user-1", r.Id.Resource)

	profile := r.GetAnnotations()
	assert.NotNil(t, profile)
}

func TestUserResource_WithFullWorker(t *testing.T) {
	user := client.User{
		ID:        "user-2",
		Username:  "bob",
		Active:    true,
		Locale:    "en-US",
		CreatedAt: "2024-06-15T12:00:00Z",
		Name: client.Name{
			DisplayName: "Bob Jones",
			GivenName:   "Bob",
			FamilyName:  "Jones",
			MiddleName:  "Michael",
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

	r, err := userResource(user, worker, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Bob Jones", r.DisplayName)
	assert.Equal(t, "user-2", r.Id.Resource)
}

func TestUserResource_WorkerWithNilNestedStructs(t *testing.T) {
	user := client.User{
		ID:        "user-3",
		Username:  "carol",
		Active:    true,
		Locale:    "en-US",
		CreatedAt: "2024-03-01T00:00:00Z",
		Name:      client.Name{DisplayName: "Carol Lee"},
		Emails:    []client.Email{{Value: "carol@example.com"}},
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

	r, err := userResource(user, worker, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Carol Lee", r.DisplayName)
}

func TestUserResource_WorkerEmptyStringsNotIncluded(t *testing.T) {
	user := client.User{
		ID:        "user-4",
		Username:  "dave",
		Active:    false,
		Locale:    "en-US",
		CreatedAt: "2024-02-01T00:00:00Z",
		Name:      client.Name{DisplayName: "Dave Kim"},
		Emails:    []client.Email{{Value: "dave@example.com"}},
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

	r, err := userResource(user, worker, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Dave Kim", r.DisplayName)
}

func TestUserResource_NoEmails(t *testing.T) {
	user := client.User{
		ID:        "user-5",
		Username:  "eve",
		Active:    true,
		Locale:    "en-US",
		CreatedAt: "2024-01-01T00:00:00Z",
		Name:      client.Name{DisplayName: "Eve Wu"},
		Emails:    nil,
	}

	r, err := userResource(user, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Eve Wu", r.DisplayName)
}

func TestUserResource_ProfileNameAndAddressFields(t *testing.T) {
	user := client.User{
		ID:        "user-7",
		Username:  "grace",
		Active:    true,
		Locale:    "en-US",
		CreatedAt: "2024-01-01T00:00:00Z",
		Name: client.Name{
			DisplayName:         "Grace Hopper",
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

	r, err := userResource(user, worker, nil)
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
		ID:        "user-8",
		Username:  "hank",
		Active:    true,
		Locale:    "en-US",
		CreatedAt: "2024-01-01T00:00:00Z",
		Name:      client.Name{DisplayName: "Hank Hill"},
		Emails:    []client.Email{{Value: "hank@example.com"}},
		Addresses: []client.Address{
			{
				Type:     "HOME",
				Locality: "Arlen",
				Region:   "TX",
				Country:  "US",
			},
		},
	}

	r, err := userResource(user, nil, nil)
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
		ID:        "user-9",
		Username:  "irene",
		Active:    true,
		Locale:    "en-US",
		CreatedAt: "2024-01-01T00:00:00Z",
		Name:      client.Name{DisplayName: "Irene Adler"},
		Emails:    []client.Email{{Value: "irene@example.com"}},
		Addresses: []client.Address{
			{
				Type: "WORK",
				// All address fields empty
			},
		},
	}

	r, err := userResource(user, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Irene Adler", r.DisplayName)
}

func TestUserResource_WithWorkLocation(t *testing.T) {
	user := client.User{
		ID:        "user-10",
		Username:  "kate",
		Active:    true,
		Locale:    "en-US",
		CreatedAt: "2024-01-01T00:00:00Z",
		Name:      client.Name{DisplayName: "Kate Walsh"},
		Emails:    []client.Email{{Value: "kate@example.com"}},
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

	r, err := userResource(user, worker, workLocation)
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
		ID:        "user-11",
		Username:  "leo",
		Active:    true,
		Locale:    "en-US",
		CreatedAt: "2024-01-01T00:00:00Z",
		Name:      client.Name{DisplayName: "Leo Park"},
		Emails:    []client.Email{{Value: "leo@example.com"}},
	}
	worker := &client.Worker{
		ID:     "worker-11",
		UserID: "user-11",
		Status: "ACTIVE",
	}

	r, err := userResource(user, worker, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Leo Park", r.DisplayName)
}

func TestUserResource_WorkLocationNameOnly(t *testing.T) {
	user := client.User{
		ID:        "user-12",
		Username:  "maya",
		Active:    true,
		Locale:    "en-US",
		CreatedAt: "2024-01-01T00:00:00Z",
		Name:      client.Name{DisplayName: "Maya Chen"},
		Emails:    []client.Email{{Value: "maya@example.com"}},
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

	r, err := userResource(user, worker, workLocation)
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
		ID:        "user-6",
		Username:  "frank",
		Active:    true,
		CreatedAt: "not-a-date",
		Name:      client.Name{DisplayName: "Frank"},
	}

	_, err := userResource(user, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse created_at")
}
