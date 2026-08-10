package main

import (
	"github.com/perplexityai/bumblebee/internal/roots"
	"github.com/perplexityai/bumblebee/internal/scanner"
)

type rootsOpts = roots.Options

func resolveRoots(profile string, explicit []string, opts rootsOpts) ([]scanner.Root, []string, error) {
	return roots.Resolve(profile, explicit, opts)
}

func classifyRoot(path, profile string) string {
	return roots.Classify(path, profile)
}

func isBroadHomeRoot(path string) bool {
	return roots.IsBroadHomeRoot(path)
}

func isLikelyUserHomeName(name string) bool {
	return roots.IsLikelyUserHomeName(name)
}

func allUsersHomes(usersDir string) []string {
	return roots.AllUsersHomes(usersDir)
}
