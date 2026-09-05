package utils

import (
	"errors"
	"log/slog"
	"regexp"
)

func GetCurrentProfile() (string, error) {
	slog.Debug("Getting current profile")
	config, err := GetConfig()
	if err != nil {
		slog.Warn("Could not get current profile")
		return "", nil
	}
	slog.Debug("Got current profile", "profile", config.Profile)
	return config.Profile, nil
}

func IsProfileExist(profile string) (bool, error) {
	slog.Debug("Checking if profile exists", "profile", profile)
	if IsEmpty(profile) {
		slog.Warn("Argument error")
		return false, ArgumentError
	}
	exists, err := IsBranchExist(RepoDir, profile)
	if err != nil {
		slog.Warn("Could not check if profile exists", "error", err)
		return false, err
	}
	slog.Debug("Checked if profile exists", "exists", exists)
	return exists, nil
}

var profileNameRegexp = regexp.MustCompile(`^(?:(?:[a-zA-Z0-9]+[\-_])*[a-zA-Z0-9]+\/)*(?:[a-zA-Z0-9]+[\-_])*[a-zA-Z0-9]+$`)

func IsValidProfileName(profile string) bool {
	slog.Debug("Checking if profile name is valid", "profile", profile)
	valid := profileNameRegexp.MatchString(profile)
	slog.Debug("Checked if profile name is valid", "valid", valid)
	return valid
}

var ErrorProfileNameNotValid = errors.New("Profile name is not valid")
var ErrorProfileAlreadyExsists = errors.New("Profile already exists")
var ErrorBaseProfileDoesNotExist = errors.New("Base profile does not exist")
var ErrorProfileDoesNotExist = errors.New("Profile does not exist")

func CreateProfile(newProfile string, baseProfile string) error {
	slog.Debug("Creating profile", "newProfile", newProfile, "baseProfile", baseProfile)
	if IsEmpty(newProfile) {
		slog.Warn("Argument is empty")
		return ArgumentError
	}

	if IsEmpty(baseProfile) {
		slog.Debug("Base profile was not provided, using current active profile.")
		var err error
		baseProfile, err = GetCurrentProfile()
		if err != nil {
			slog.Warn("Could not get current active profile")
			return err
		}
	}

	valid := IsValidProfileName(newProfile)
	if !valid {
		slog.Warn("Profile name is not valid")
		return ErrorProfileNameNotValid
	}

	newExists, err := IsProfileExist(newProfile)
	if err != nil {
		slog.Warn("Could not check if new profile exists")
		return err
	}
	if newExists {
		slog.Warn("Profile with that name already exists")
		return ErrorProfileAlreadyExsists
	}

	baseExists, err := IsProfileExist(baseProfile)
	if err != nil {
		slog.Warn("Could not check if base profile exists")
		return err
	}
	if !baseExists {
		slog.Warn("Base profile does not exist.")
		return ErrorBaseProfileDoesNotExist
	}

	err = CreateBranchWithBase(RepoDir, newProfile, baseProfile)
	if err != nil {
		slog.Warn("Could not create profile branch", "error", err)
		return err
	}

	slog.Debug("Created new profile")
	return nil
}

func GetProfiles() ([]string, error) {
	slog.Debug("Getting profiles")

	branches, err := GetBranches(RepoDir)
	if err != nil {
		slog.Warn("Failed to get branches", "error", err)
		return nil, err
	}

	slog.Debug("Got profiles", "profiles", branches)
	return branches, nil
}

func RemoveProfile(profile string) error {
	slog.Debug("Removing profile", "profile", profile)
	if IsEmpty(profile) {
		slog.Warn("Argument error")
		return ArgumentError
	}
	err := RemoveBranch(RepoDir, profile)
	if err != nil {
		slog.Warn("Failed to remove profile", "error", err)
		return err
	}
	slog.Debug("Removed profile")
	return err
}

func SetActiveProfile(profile string) error {
	slog.Debug("Setting active profile", "profile", profile)
	if IsEmpty(profile) {
		slog.Warn("Argument error")
		return ArgumentError
	}

	config, err := GetConfig()
	if err != nil {
		slog.Warn("Couldn't get config", "error", err)
		return err
	}

	config.Profile = profile

	err = UpdateConfig(config)
	if err != nil {
		slog.Warn("Could not update config.")
		return err
	}

	slog.Debug("Set active profile to profile", "profile", profile)
	return nil
}

func RenameProfile(oldName string, newName string) error {
	slog.Debug("Renaming profile", "oldName", oldName, "newName", newName)

	if IsEmpty(oldName) {
		slog.Warn("Argument error : oldName")
		return ArgumentError
	}

	if IsEmpty(newName) {
		slog.Warn("Argument error : newName")
		return ArgumentError
	}

	existsOld, err := IsProfileExist(oldName)
	if err != nil {
		slog.Warn("Couldn't check if old profile exists")
		return err
	}

	if !existsOld {
		slog.Warn("Profile with that name does not exist")
		return ErrorProfileDoesNotExist
	}

	existsNew, err := IsProfileExist(newName)
	if err != nil {
		slog.Warn("Couldn't check if new profile exists")
		return err
	}

	if existsNew {
		slog.Warn("Profile with that new already exists")
		return ErrorProfileAlreadyExsists
	}

	if IsValidProfileName(newName) {
		slog.Warn("Profile name is not valid")
		return ErrorProfileNameNotValid
	}

	err = ExecNoOutput("git", "branch", "-m", oldName, newName)
	if err != nil {
		slog.Warn("Command failed", "error", err)
		return err
	}

	slog.Debug("Renamed branch")
	return nil
}
