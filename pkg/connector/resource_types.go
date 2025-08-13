package connector

import (
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
)

var (
	// The user resource type is for all user objects from the database.
	userResourceType = &v2.ResourceType{
		Id:          "user",
		DisplayName: "User",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_USER},
	}
	organizationResourceType = &v2.ResourceType{
		Id:          "organization",
		DisplayName: "Organization",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_GROUP},
	}
	roleResourceType = &v2.ResourceType{
		Id:          "role",
		DisplayName: "Role",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_ROLE},
	}

	resourceServerResourceType = &v2.ResourceType{
		Id:          "resource_server",
		DisplayName: "Resource Server",
		Traits:      []v2.ResourceType_Trait{},
	}

	scopeResourceType = &v2.ResourceType{
		Id:          "scope",
		DisplayName: "Scope",
		Traits:      []v2.ResourceType_Trait{},
	}
)
