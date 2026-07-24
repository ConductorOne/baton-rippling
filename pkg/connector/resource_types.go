package connector

import (
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
)

// TeamResourceTypeID is the resource type ID for teams, exported so callers
// (e.g. main.go) can check it against WillSyncResourceType without a magic string.
const TeamResourceTypeID = "team"

// The user resource type is for all user objects from the database.
var userResourceType = &v2.ResourceType{
	Id:          "user",
	DisplayName: "User",
	Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_USER},
	Annotations: annotations.New(&v2.SkipEntitlements{}),
}

var teamResourceType = &v2.ResourceType{
	Id:          TeamResourceTypeID,
	DisplayName: "Team",
	Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_GROUP},
	Annotations: annotations.New(&v2.SkipGrants{}),
}
