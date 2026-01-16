package main

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("62")  // Purple
	secondaryColor = lipgloss.Color("241") // Gray
	accentColor    = lipgloss.Color("86")  // Cyan
	errorColor     = lipgloss.Color("196") // Red
	successColor   = lipgloss.Color("82")  // Green
	warningColor   = lipgloss.Color("214") // Orange

	// Title style
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	// Status bar style
	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	// Selected item style (highlighted bar)
	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("62"))

	// Normal item style
	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	// Directory style
	dirStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor)

	// File style
	fileStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	// Cursor style
	cursorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor)

	// Checked item style (selected for queue)
	checkedStyle = lipgloss.NewStyle().
			Foreground(successColor)

	// Success style
	successStyle = lipgloss.NewStyle().
			Foreground(successColor)

	// Help style
	helpStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			MarginTop(1)

	// Error style
	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	// Progress bar styles
	progressBarStyle = lipgloss.NewStyle().
				Foreground(accentColor)

	progressCompleteStyle = lipgloss.NewStyle().
				Foreground(successColor)

	// Queue item style
	queueItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	// Header style
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(secondaryColor).
			MarginBottom(1)

	// Filter input style
	filterPromptStyle = lipgloss.NewStyle().
				Foreground(accentColor)

	filterTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	// Size style
	sizeStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Width(10).
			Align(lipgloss.Right)

	// Spinner style
	spinnerStyle = lipgloss.NewStyle().
			Foreground(accentColor)
)
