package server

const (
	ModeVariable = "NWM_MODE"
	ModeDev      = "dev"
	ModeProd     = "prod"
)

func ApplyMode(directory string, mode string) ([]string, error) {
	config, err := LoadConfig(directory)
	if err != nil {
		return nil, err
	}
	if err := config.Set("custom_settings."+ModeVariable, mode); err != nil {
		return nil, err
	}
	if err := config.Save(); err != nil {
		return nil, err
	}
	return []string{ModeVariable + "=" + mode}, nil
}
