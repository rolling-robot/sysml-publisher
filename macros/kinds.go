package macros

import (
	"context"
	"strings"

	"github.com/Open-MBEE/OpenSysML/client/opensysml"

	"github.com/rolling-robot/sysml-publisher/internal/model"
)

// sysmlKinds are the SysML metaclasses an element can be written as, as the
// SysML v2 API spells an element's @type. Copied from OpenSysML's export tables
// (internal/translate/export), which are not public API; kinds that occur
// in the model are accepted too, so a kind missing here still works when
// the model uses it.
var sysmlKinds = []string{
	"AcceptActionUsage", "ActionDefinition", "ActionUsage",
	"AllocationDefinition", "AllocationUsage", "AnalysisCaseDefinition",
	"AnalysisCaseUsage", "AssertConstraintUsage", "AssignmentActionUsage",
	"AssociationUsage", "AttributeDefinition", "AttributeUsage",
	"BehaviorUsage", "BindingConnectorAsUsage", "BindingConnectorDefinition",
	"BooleanUsage", "CalculationDefinition", "CalculationUsage",
	"CaseDefinition", "CaseUsage", "ClassUsage", "ConcernDefinition",
	"ConcernUsage", "ConjugatedPortDefinition", "ConnectionDefinition",
	"ConnectionUsage", "ConnectorAsUsage", "ConstraintDefinition",
	"ConstraintUsage", "EnumerationDefinition", "EnumerationUsage",
	"EventOccurrenceUsage", "ExhibitStateUsage", "FlowDefinition", "FlowUsage",
	"ForLoopActionUsage", "IfActionUsage", "IncludeUseCaseUsage",
	"IndividualDefinition", "IndividualUsage", "InteractionUsage",
	"InterfaceDefinition", "InterfaceUsage", "ItemDefinition", "ItemUsage",
	"LibraryPackage", "MetadataDefinition", "MetadataUsage",
	"OccurrenceDefinition", "OccurrenceUsage", "Package", "PartDefinition",
	"PartUsage", "PerformActionUsage", "PortDefinition", "PortUsage",
	"PredicateUsage", "ReferenceUsage", "RenderingDefinition",
	"RenderingUsage", "RequirementDefinition", "RequirementUsage",
	"SatisfyRequirementUsage", "SendActionUsage", "StateDefinition",
	"StateUsage", "StructureUsage", "SuccessionAsUsage", "SuccessionFlowUsage",
	"TerminateActionUsage", "TransitionUsage", "UseCaseDefinition",
	"UseCaseUsage", "VerificationCaseDefinition", "VerificationCaseUsage",
	"ViewDefinition", "ViewUsage", "ViewpointDefinition", "ViewpointUsage",
	"WhileLoopActionUsage",
}

// kindIndex finds the element kind a macro names: a metaclass name in any
// letter case (partUsage, PartUsage).
type kindIndex map[string]string // lower-case name → metaclass

func newKindIndex(ctx context.Context, m *model.Model) (kindIndex, error) {
	k := kindIndex{}
	for _, kind := range sysmlKinds {
		k[strings.ToLower(kind)] = kind
	}
	all, err := m.Query(ctx, opensysml.Query{Select: []string{"@type"}})
	if err != nil {
		return nil, err
	}
	for _, e := range all {
		if e.Type != "" {
			k[strings.ToLower(e.Type)] = e.Type
		}
	}
	return k, nil
}

// kinds is the kind index, made on first use: only sysml-members-list with
// a kind needs it, and it queries the whole model.
func (e *env) kinds() (kindIndex, error) {
	if e.kindIndex == nil {
		k, err := newKindIndex(e.ctx, e.m)
		if err != nil {
			return nil, err
		}
		e.kindIndex = k
	}
	return e.kindIndex, nil
}

// lookup is the metaclass name refers to, and whether it is one.
func (k kindIndex) lookup(name string) (string, bool) {
	kind, ok := k[strings.ToLower(name)]
	return kind, ok
}
