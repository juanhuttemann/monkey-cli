package tui

// Primary brand colors.
// ColorPrimary is the anchor color for persistent UI elements (e.g. input border).
// ColorAccent is the warm highlight used for active/selected states.
const (
	ColorPrimary = "#461914" // Deep Brown-Red
	ColorAccent  = "#A89228" // Antique Gold
)

// Message role border colors.
const (
	ColorUserBorder      = "#8B3A21" // Rust Brown    — warm, personal
	ColorAssistantBorder = "#729B2F" // Leaf Green    — natural, positive
	ColorToolBorder      = "#225057" // Dark Slate    — cool, functional
	ColorSystemBorder    = "#305322" // Forest Green  — subdued, informational
	ColorErrorBorder     = "#BA3F28" // Burnt Orange  — danger/warning
)

// Muted gray scale — explicit names instead of ad-hoc hex literals.
const (
	ColorGrayLight = "#AAAAAA" // timer, secondary info
	ColorGrayMid   = "#888888" // muted labels, canceled state
	ColorGrayDark  = "#666666" // picker borders, subdued help text
	ColorGrayDeep  = "#555555" // timestamps, inactive state, faint text
)

// ColorPickerCursor is the highlighted-row color in all pickers.
// Uses ColorAccent so selection reads as "warm highlight", not "assistant message".
const ColorPickerCursor = ColorAccent

// Diff view colors — derived from the app's existing semantic palette.
// Using ColorAssistantBorder and ColorToolBorder eliminates the three-greens
// and GitHub-blue problems: there is now one green and one cyan in the whole app.
const (
	ColorDiffAdd    = ColorAssistantBorder // green — positive change, same as assistant border
	ColorDiffDel    = ColorErrorBorder     // red — destructive change, same as error border
	ColorDiffHunk   = ColorToolBorder      // cyan — structural/informational, same as tool border
	ColorDiffGutter = ColorGrayDeep        // faint — line numbers recede into the background
)

// Tool approval dialog colors, derived from existing semantic tokens.
const (
	ColorApprovalModel = ColorAssistantBorder // green — the assistant is acting
	ColorApprovalTool  = ColorToolBorder      // cyan — it is a tool call
)

// Intro art pixel colors — four tiers of block shading for the monkey sprite.
const (
	ColorMonkeyDark  = "#2B1200" // very dark brown  — ██ outline
	ColorMonkeyMid   = "#7B4220" // medium brown     — ▒▒ body
	ColorMonkeySkin  = "#7c3400" // warm brown       — ▓▓ skin
	ColorMonkeyLight = "#C49A6C" // warm tan         — ░░ face / underbelly
)
