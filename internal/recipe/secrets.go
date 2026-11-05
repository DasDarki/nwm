package recipe

import (
	"fmt"

	"github.com/DasDarki/nwm/internal/secrets"
)

type PromptFunc func(name string) (string, error)

func EnsureSecrets(manifest *Manifest, store *secrets.Store, provided map[string]string, prompt PromptFunc) ([]string, error) {
	var created []string
	for _, secret := range manifest.Secrets {
		if value, given := provided[secret.Name]; given {
			store.Set(secret.Name, value)
			continue
		}
		if _, exists := store.Get(secret.Name); exists {
			continue
		}
		var value string
		var err error
		if secret.Prompt {
			if prompt == nil {
				return created, fmt.Errorf("secret %q needs a value, pass --secret %s=...", secret.Name, secret.Name)
			}
			value, err = prompt(secret.Name)
			if err == nil && value == "" {
				err = fmt.Errorf("secret %q cannot be empty", secret.Name)
			}
		} else {
			value, err = secrets.Random(secret.Length)
		}
		if err != nil {
			return created, err
		}
		store.Set(secret.Name, value)
		created = append(created, secret.Name)
	}
	return created, nil
}
