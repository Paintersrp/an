package constants

const (
	Version        = `0.0.1`
	ConfigFile     = `cfg`
	ConfigFileType = `yaml`
	ConfigDir      = `/.zet-cli/`

	Help = `Usage:
	{{if .Runnable}}
		{{.UseLine}}
	{{end}}
	{{if .HasAvailableSubCommands}}
		{{.CommandPath}} [command]
	{{end}}
	{{if gt (len .Aliases) 0}}
  	Aliases:
		{{.NameAndAliases}}
	{{end}}
  {{if .HasExample}}
  Examples:
	{{.Example}}
  {{end}}
  {{if .HasAvailableSubCommands}}
  Available Commands:
	{{range .Commands}}
		{{if (or .IsAvailableCommand (eq .Name "help"))}}
			{{rpad .Name .NamePadding }} {{.Short}}
		{{end}}
	{{end}}
  {{end}}
  {{if .HasAvailableLocalFlags}}
  Flags:
  {{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
  {{end}}
  {{if .HasAvailableInheritedFlags}}
  Global Flags:
  {{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}
  {{end}}
  {{if .HasHelpSubCommands}}
  Additional help topics:
	{{range .Commands}}
		{{if .IsAdditionalHelpTopicCommand}}
		{{rpad .CommandPath .CommandPathPadding}} {{.Short}}
		{{end}}
	{{end}}
  {{end}}
  {{if .HasAvailableSubCommands}}
  Use "{{.CommandPath}} [command] --help" for more information about a command.
  {{end}}
  `
)
