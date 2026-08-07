package connector

import (
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
)

// The user resource type is for all user objects from the database.
// newUserBuilder clones this and adds SkipEntitlements, or
// SkipEntitlementsAndGrants when team isn't synced.
var userResourceType = &v2.ResourceType{
	Id:          "user",
	DisplayName: "User",
	Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_USER},
}

// TeamResourceTypeID is referenced when gating cross-type team grants.
const TeamResourceTypeID = "team"

var teamResourceType = &v2.ResourceType{
	Id:          TeamResourceTypeID,
	DisplayName: "Team",
	Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_GROUP},
	Annotations: annotations.New(&v2.SkipGrants{}),
}
