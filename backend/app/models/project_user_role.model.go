package models

import (
	"github.com/google/uuid"
)

type MemberProjectRole struct {
	ProjectId uuid.UUID `json:"projectId" lit:"project_id"`
	Name      string    `json:"name" lit:"name"`
	Framework string    `json:"framework" lit:"framework"`
	Role      *string   `json:"role" lit:"role"`
}
