package skill

import "github.com/jguan/ai-inference-managed-by-ai/pkg/unit"

// Skill domain errors.
var (
	ErrSkillNotFound         = unit.NewDomainError("skill", unit.ErrCodeSkillNotFound, "skill not found")
	ErrSkillAlreadyExists    = unit.NewDomainError("skill", unit.ErrCodeSkillAlreadyExists, "skill already exists")
	ErrSkillInvalid          = unit.NewDomainError("skill", unit.ErrCodeSkillInvalid, "skill is invalid")
	ErrBuiltinSkillImmutable = unit.NewDomainError("skill", unit.ErrCodeBuiltinSkillImmutable, "builtin skill cannot be modified")

	ErrInvalidInput = unit.NewError(unit.ErrCodeInvalidInput, "invalid input")
)
