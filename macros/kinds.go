package macros

import (
	"strings"

	"github.com/rolling-robot/sysml-publisher/internal/model"
)

// sysmlKinds are the SysML metaclasses an element can be written as, as the
// API JSON spells its @type. Copied from OpenSysML's export tables
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

func newKindIndex(m *model.Model) kindIndex {
	k := kindIndex{}
	for _, kind := range sysmlKinds {
		k[strings.ToLower(kind)] = kind
	}
	for _, id := range m.Order {
		if kind := m.Elements[id].Type(); kind != "" {
			k[strings.ToLower(kind)] = kind
		}
	}
	return k
}

// lookup is the metaclass name refers to, and whether it is one.
func (k kindIndex) lookup(name string) (string, bool) {
	kind, ok := k[strings.ToLower(name)]
	return kind, ok
}
