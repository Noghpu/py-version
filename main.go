package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// Version represents a semantic version with major, minor, and patch components
type Version struct {
	Major int
	Minor int
	Patch int
}

func parseVersion(versionStr string) (Version, error) {
	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)$`)
	matches := re.FindStringSubmatch(versionStr)
	
	if matches == nil || len(matches) != 4 {
		return Version{}, fmt.Errorf("invalid version format: %s", versionStr)
	}
	
	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])
	
	return Version{Major: major, Minor: minor, Patch: patch}, nil
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func findProjectFiles(specifiedFiles []string) ([]string, error) {
	if len(specifiedFiles) > 0 {
		return specifiedFiles, nil
	}
	
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %v", err)
	}
	
	defaultFile := filepath.Join(currentDir, "pyproject.toml")
	if _, err := os.Stat(defaultFile); err == nil {
		return []string{defaultFile}, nil
	}
	
	return nil, fmt.Errorf("no pyproject.toml found in current directory")
}

func updateVersion(filePath string, updateFunc func(Version) Version) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %v", filePath, err)
	}
	
	fileContent := string(content)
	
	re := regexp.MustCompile(`(version\s*=\s*["'])(\d+\.\d+\.\d+)(["'])`)
	match := re.FindStringSubmatch(fileContent)
	
	if match == nil || len(match) != 4 {
		return fmt.Errorf("version not found in %s", filePath)
	}
	
	oldVersion, err := parseVersion(match[2])
	if err != nil {
		return err
	}
	
	newVersion := updateFunc(oldVersion)
	
	updatedContent := re.ReplaceAllString(
		fileContent, 
		fmt.Sprintf("${1}%s${3}", newVersion.String()),
	)
	
	err = os.WriteFile(filePath, []byte(updatedContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %v", filePath, err)
	}
	
	fmt.Printf("Updated %s: %s → %s\n", filePath, oldVersion.String(), newVersion.String())
	return nil
}

func validateComponent(component string) error {
	validComponents := map[string]bool{
		"major": true,
		"minor": true,
		"patch": true,
	}
	
	if !validComponents[strings.ToLower(component)] {
		return fmt.Errorf("invalid component: %s (must be 'major', 'minor', or 'patch')", component)
	}
	
	return nil
}

func main() {
	var files []string
	var amount int
	
	rootCmd := &cobra.Command{
		Use:   "py-version",
		Short: "A tool to manage version numbers in pyproject.toml files",
	}
	
	incrementCmd := &cobra.Command{
		Use:   "increment [component]",
		Short: "Increment a version component (major, minor, or patch)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			component := strings.ToLower(args[0])
			
			if err := validateComponent(component); err != nil {
				return err
			}
			
			projectFiles, err := findProjectFiles(files)
			if err != nil {
				return err
			}
			
			updateFunc := func(v Version) Version {
				switch component {
				case "major":
					v.Major += amount
					v.Minor = 0
					v.Patch = 0
				case "minor":
					v.Minor += amount
					v.Patch = 0
				case "patch":
					v.Patch += amount
				}
				return v
			}
			
			for _, file := range projectFiles {
				if err := updateVersion(file, updateFunc); err != nil {
					return err
				}
			}
			
			return nil
		},
	}
	
	decrementCmd := &cobra.Command{
		Use:   "decrement [component]",
		Short: "Decrement a version component (major, minor, or patch)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			component := strings.ToLower(args[0])
			
			if err := validateComponent(component); err != nil {
				return err
			}
			
			projectFiles, err := findProjectFiles(files)
			if err != nil {
				return err
			}
			
			updateFunc := func(v Version) Version {
				switch component {
				case "major":
					v.Major = max(0, v.Major-amount)
				case "minor":
					v.Minor = max(0, v.Minor-amount)
				case "patch":
					v.Patch = max(0, v.Patch-amount)
				}
				return v
			}
			
			for _, file := range projectFiles {
				if err := updateVersion(file, updateFunc); err != nil {
					return err
				}
			}
			
			return nil
		},
	}
	
	setCmd := &cobra.Command{
		Use:   "set [component] [value]",
		Short: "Set a version component to a specific value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			component := strings.ToLower(args[0])
			
			if err := validateComponent(component); err != nil {
				return err
			}
			
			value, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid value: %s", args[1])
			}
			
			if value < 0 {
				return fmt.Errorf("version components cannot be negative")
			}
			
			projectFiles, err := findProjectFiles(files)
			if err != nil {
				return err
			}
			
			updateFunc := func(v Version) Version {
				switch component {
				case "major":
					v.Major = value
				case "minor":
					v.Minor = value
				case "patch":
					v.Patch = value
				}
				return v
			}
			
			for _, file := range projectFiles {
				if err := updateVersion(file, updateFunc); err != nil {
					return err
				}
			}
			
			return nil
		},
	}
	
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Display the current version without modifying it",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectFiles, err := findProjectFiles(files)
			if err != nil {
				return err
			}
			
			for _, file := range projectFiles {
				content, err := os.ReadFile(file)
				if err != nil {
					return fmt.Errorf("failed to read file %s: %v", file, err)
				}
				
				re := regexp.MustCompile(`version\s*=\s*["'](\d+\.\d+\.\d+)["']`)
				match := re.FindStringSubmatch(string(content))
				
				if match == nil || len(match) != 2 {
					return fmt.Errorf("version not found in %s", file)
				}
				
				fmt.Printf("%s: %s\n", file, match[1])
			}
			
			return nil
		},
	}
	
	// Add flags
	incrementCmd.Flags().StringSliceVarP(&files, "files", "f", nil, "Files to update (default: pyproject.toml in current dir)")
	incrementCmd.Flags().IntVar(&amount, "amount", 1, "Amount to increment by")
	
	decrementCmd.Flags().StringSliceVarP(&files, "files", "f", nil, "Files to update (default: pyproject.toml in current dir)")
	decrementCmd.Flags().IntVar(&amount, "amount", 1, "Amount to decrement by")
	
	setCmd.Flags().StringSliceVarP(&files, "files", "f", nil, "Files to update (default: pyproject.toml in current dir)")
	
	showCmd.Flags().StringSliceVarP(&files, "files", "f", nil, "Files to show version from (default: pyproject.toml in current dir)")
	
	rootCmd.AddCommand(incrementCmd, decrementCmd, setCmd, showCmd)
	
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
