package displaylist

// Role is a semantic tag attached to every Item. Emitters look up
// colors, fonts, and line widths in a caller-supplied per-Role style
// map. Standard roles are defined as constants below; callers may use
// custom Role values, in which case emitters fall back to the default
// style.
type Role string

const (
	RoleUnknown Role = ""

	RoleNode             Role = "node"
	RoleEdge             Role = "edge"
	RoleEdgeLabel        Role = "edgeLabel"
	RoleSubgraph         Role = "subgraph"
	RoleClusterTitle     Role = "clusterTitle"
	RolePseudostateStart Role = "pseudostateStart"
	RolePseudostateEnd   Role = "pseudostateEnd"
	RoleStateBox         Role = "stateBox"
	RoleStateComposite   Role = "stateComposite"

	RoleActorBox      Role = "actorBox"
	RoleActorTitle    Role = "actorTitle"
	RoleLifeline      Role = "lifeline"
	RoleActivation    Role = "activation"
	RoleMessageLabel  Role = "messageLabel"
	RoleNoteText      Role = "noteText"
	RoleSequenceNote  Role = "sequenceNote"
	RoleLoopBlock     Role = "loopBlock"
	RoleAltBlock      Role = "altBlock"
	RoleOptBlock      Role = "optBlock"
	RoleParBlock      Role = "parBlock"
	RoleCriticalBlock Role = "criticalBlock"
	RoleBreakBlock    Role = "breakBlock"
	RoleRectBlock     Role = "rectBlock"

	RoleClassBox        Role = "classBox"
	RoleClassMember     Role = "classMember"
	RoleClassAnnotation Role = "classAnnotation"

	RoleEntityBox       Role = "entityBox"
	RoleEntityAttribute Role = "entityAttribute"
)
