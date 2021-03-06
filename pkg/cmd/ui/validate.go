package ui

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/rhd-gitops-example/gitops-cli/pkg/cmd/utility"
	"github.com/rhd-gitops-example/gitops-cli/pkg/pipelines/git"
	"github.com/rhd-gitops-example/gitops-cli/pkg/pipelines/ioutils"
	"github.com/rhd-gitops-example/gitops-cli/pkg/pipelines/secrets"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/klog"
)

func makePrefixValidator() survey.Validator {
	return func(input interface{}) error {
		return validatePrefix(input)
	}
}

func makeSecretValidator() survey.Validator {
	return func(input interface{}) error {
		return validateSecretLength(input)
	}
}

func makeOverWriteValidator(path string) survey.Validator {
	return func(input interface{}) error {
		return validateOverwriteOption(input, path)
	}
}

func makeSealedSecretsService(sealedSecretService *types.NamespacedName) survey.Validator {
	return func(input interface{}) error {
		return validateSealedSecretService(input, sealedSecretService)
	}
}

func makeAccessTokenCheck(serviceRepo string) survey.Validator {
	return func(input interface{}) error {
		return validateAccessToken(input, serviceRepo)
	}
}

// ValidatePrefix checks the length of the prefix with the env crosses 63 chars or not
func validatePrefix(input interface{}) error {
	if s, ok := input.(string); ok {
		prefix := utility.MaybeCompletePrefix(s)
		s = prefix + "stage"
		if len(s) < 64 {
			err := ValidateName(s)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("The prefix %s, must be less than 58 characters", prefix)
		}
		return nil
	}
	return nil
}

// ValidateName will do validation of application & component names according to DNS (RFC 1123) rules
// Criteria for valid name in kubernetes: https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/identifiers.md
func ValidateName(name string) error {

	errorList := validation.IsDNS1123Label(name)

	if len(errorList) != 0 {
		return fmt.Errorf("%s is not a valid name:  %s", name, strings.Join(errorList, " "))
	}

	return nil
}

func validateSecretLength(input interface{}) error {
	if s, ok := input.(string); ok {
		err := CheckSecretLength(s)
		if err {
			return fmt.Errorf("The secret length should 16 or more ")
		}
		return nil
	}
	return nil
}

// validateOverwriteOption(  validates the URL
func validateOverwriteOption(input interface{}, path string) error {
	if s, ok := input.(string); ok {
		if s == "no" {
			exists, _ := ioutils.IsExisting(ioutils.NewFilesystem(), filepath.Join(path, "pipelines.yaml"))
			if exists {
				EnterOutputPath()
			}
		}
		return nil
	}
	return nil

}

// validateAccessToken validates if the access token is correct for a particular service repo
func validateAccessToken(input interface{}, serviceRepo string) error {
	if s, ok := input.(string); ok {
		repo, err := git.NewRepository(serviceRepo, s)
		if err != nil {
			return err
		}
		parsedURL, err := url.Parse(serviceRepo)
		if err != nil {
			return fmt.Errorf("failed to parse the provided URL %q: %w", serviceRepo, err)
		}
		repoName, err := git.GetRepoName(parsedURL)
		if err != nil {
			return fmt.Errorf("failed to get the repository name from %q: %w", serviceRepo, err)
		}
		_, _, err = repo.Client.Repositories.Find(context.Background(), repoName)
		if err != nil {
			return fmt.Errorf("The token passed is incorrect for repository %s", repoName)
		}
		return nil
	}
	return nil
}

// validateSealedSecretService validates to see if the sealed secret service is present in the correct namespace.
func validateSealedSecretService(input interface{}, sealedSecretService *types.NamespacedName) error {
	if s, ok := input.(string); ok {
		sealedSecretService.Name = s
		sealedSecretService.Namespace = EnterSealedSecretNamespace()
		_, err := secrets.GetClusterPublicKey(*sealedSecretService)
		if err != nil {
			if compareError(err, sealedSecretService.Name) {
				return fmt.Errorf("The given service %q is not installed in the right namespace %q", sealedSecretService.Name, sealedSecretService.Namespace)
			}
			return errors.New("sealed secrets could not be configured sucessfully")
		}
		return nil
	}
	return nil
}

func compareError(err error, sealedSecretService string) bool {
	createdError := fmt.Errorf("cannot fetch certificate: services \"%s\" not found", sealedSecretService)
	return err.Error() == createdError.Error()
}

// check if the length of secret is less than 16 chars
func CheckSecretLength(secret string) bool {
	if secret != "" {
		if len(secret) < 16 {
			return true
		}
	}
	return false
}

// handleError handles UI-related errors, in particular useful to gracefully handle ctrl-c interrupts gracefully
func handleError(err error) {
	if err != nil {
		if err == terminal.InterruptErr {
			os.Exit(1)
		} else {
			klog.V(4).Infof("Encountered an error processing prompt: %v", err)
		}
	}
}
