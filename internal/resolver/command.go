// SPDX-FileCopyrightText: 2023 Iván SZKIBA
//
// SPDX-License-Identifier: AGPL-3.0-only

//nolint:revive,gochecknoglobals
package resolver

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"

	"github.com/szkiba/k6x/internal/dependency"
)

var (
	reK6        = regexp.MustCompile(`k6 (?P<k6Version>[^ ]+) .*`)
	reExtension = regexp.MustCompile(
		`  (?P<extModule>[^ ]+) (?P<extVersion>[^,]+), (?P<extName>[^ ]+) \[([^\]]+)\]`,
	)
	idxK6Version  = reK6.SubexpIndex("k6Version")
	idxExtModule  = reExtension.SubexpIndex("extModule")
	idxExtVersion = reExtension.SubexpIndex("extVersion")
	idxExtName    = reExtension.SubexpIndex("extName")
)

type commandResolver struct {
	cmd  string
	args []string
}

func FromCommand(cmd string, args ...string) Resolver {
	return &commandResolver{cmd: cmd, args: args}
}

func (res *commandResolver) Resolve(
	ctx context.Context,
	deps dependency.Dependencies,
) (Ingredients, error) {
	out, err := exec.CommandContext(ctx, res.cmd, res.args...).Output() //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrResolver, err.Error())
	}

	return res.resolveFromOutput(out, deps)
}

func (res *commandResolver) resolveFromOutput(
	out []byte,
	deps dependency.Dependencies,
) (Ingredients, error) {
	ings, err := parseCommandOutput(out)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrResolver, err.Error())
	}

	return ings.filter(deps), nil
}

func CommandDependencies(
	ctx context.Context,
	cmd string,
	args ...string,
) (dependency.Dependencies, error) {
	out, err := exec.CommandContext(ctx, cmd, args...).Output()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrResolver, err.Error())
	}

	ings, err := parseCommandOutput(out)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrResolver, err.Error())
	}

	deps := make(dependency.Dependencies)

	for name := range ings {
		deps[name] = &dependency.Dependency{Name: name}
	}

	return deps, nil
}

func parseCommandOutput(text []byte) (Ingredients, error) {
	var err error
	var ing *Ingredient

	ings := make(Ingredients)

	if allmatch := reK6.FindAllSubmatch(text, -1); allmatch != nil {
		match := allmatch[0]

		ing, err = NewIngredient(k6, string(match[idxK6Version]), "")
		if err != nil {
			return nil, err
		}

		ings[ing.Name] = ing
	}

	for _, match := range reExtension.FindAllSubmatch(text, -1) {
		ing, err = NewIngredient(
			string(match[idxExtName]),
			string(match[idxExtVersion]),
			string(match[idxExtModule]),
		)
		if err != nil {
			return nil, err
		}

		ings[ing.Name] = ing
	}

	return ings, nil
}

const k6 = "k6"
