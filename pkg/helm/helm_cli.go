package helm

import (
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
)

// HelmCLI implements common helm actions based on helm CLI
type HelmCLI struct {
	Binary     string
	BinVersion Version
	CWD        string
	Runner     *util.Command
}

// NewHelmCLI creates a new HelmCLI instance configured to used the provided helm CLI in
// the given current working directory
func NewHelmCLI(binary string, version Version, cwd string, args ...string) *HelmCLI {
	a := []string{}
	for _, x := range args {
		y := strings.Split(x, " ")
		for _, z := range y {
			a = append(a, z)
		}
	}
	runner := &util.Command{
		Args: a,
		Name: binary,
		Dir:  cwd,
	}
	cli := &HelmCLI{
		Binary:     binary,
		BinVersion: version,
		CWD:        cwd,
		Runner:     runner,
	}
	return cli
}

// SetCWD configures the common working directory of helm CLI
func (h *HelmCLI) SetCWD(dir string) {
	h.CWD = dir
}

// HelmBinary return the configured helm CLI
func (h *HelmCLI) HelmBinary() string {
	return h.Binary
}

// SetHelmBinary configure a new helm CLI
func (h *HelmCLI) SetHelmBinary(binary string) {
	h.Binary = binary
}

func (h *HelmCLI) runHelm(args ...string) error {
	h.Runner.Name = h.Binary
	h.Runner.Dir = h.CWD
	h.Runner.Args = args
	_, err := h.Runner.RunWithoutRetry()
	return err
}

func (h *HelmCLI) runHelmWithOutput(args ...string) (string, error) {
	h.Runner.Dir = h.CWD
	h.Runner.Name = h.Binary
	h.Runner.Args = args
	return h.Runner.RunWithoutRetry()
}

// Init executes the helm init command according with the given flags
func (h *HelmCLI) Init(clientOnly bool, serviceAccount string, tillerNamespace string, upgrade bool) error {
	args := []string{}
	args = append(args, "init")
	if clientOnly {
		args = append(args, "--client-only")
	}
	if serviceAccount != "" {
		args = append(args, "--service-account", serviceAccount)
	}
	if tillerNamespace != "" {
		args = append(args, "--tiller-namespace", tillerNamespace)
	}
	if upgrade {
		args = append(args, "--upgrade")
	}
	return h.runHelm(args...)
}

// AddRepo adds a new helm repo with the given name and URL
func (h *HelmCLI) AddRepo(repo string, URL string) error {
	return h.runHelm("repo", "add", repo, URL)
}

// RemoveRepo removes the given repo from helm
func (h *HelmCLI) RemoveRepo(repo string) error {
	return h.runHelm("repo", "remove", repo)
}

// ListRepos list the installed helm repos together with their URL
func (h *HelmCLI) ListRepos() (map[string]string, error) {
	output, err := h.runHelmWithOutput("repo", "list")
	if err != nil {
		return nil, errors.Wrap(err, "failed to list repositories")
	}
	repos := map[string]string{}
	lines := strings.Split(output, "\n")
	for _, line := range lines[2:] {
		line = strings.TrimSpace(line)
		fields := strings.Fields(line)
		if len(fields) > 1 {
			repos[fields[0]] = fields[1]
		} else if len(fields) > 0 {
			repos[fields[0]] = ""
		}
	}
	return repos, nil
}

// IsRepoMissing checks if the repository with the given URL is missing from helm
func (h *HelmCLI) IsRepoMissing(URL string) (bool, error) {
	repos, err := h.ListRepos()
	if err != nil {
		return true, errors.Wrap(err, "failed to list the repositories")
	}
	searchedURL, err := url.Parse(URL)
	if err != nil {
		return true, errors.Wrap(err, "provided repo URL is invalid")
	}
	for _, repoURL := range repos {
		if len(repoURL) > 0 {
			url, err := url.Parse(repoURL)
			if err != nil {
				return true, errors.Wrap(err, "failed to parse the repo URL")
			}
			if url.Host == searchedURL.Host {
				return false, nil
			}
		}
	}
	return true, nil
}

// UpdateRepo updates the helm repositories
func (h *HelmCLI) UpdateRepo() error {
	return h.runHelm("repo", "update")
}

// RemoveRequirementsLock removes the requirements.lock file from the current working directory
func (h *HelmCLI) RemoveRequirementsLock() error {
	dir := h.CWD
	path := filepath.Join(dir, "requirements.lock")
	exists, err := util.FileExists(path)
	if err != nil {
		return errors.Wrapf(err, "no requirements.lock file found in directory '%s'", dir)
	}
	if exists {
		err = os.Remove(path)
		if err != nil {
			return errors.Wrap(err, "failed to remove the requirements.lock file")
		}
	}
	return nil
}

// BuildDependency builds the helm dependencies of the helm chart from the current working directory
func (h *HelmCLI) BuildDependency() error {
	return h.runHelm("dependency", "build")
}

// InstallChart installs a helm chart according with the given flags
func (h *HelmCLI) InstallChart(chart string, releaseName string, ns string, version *string, timeout *int,
	values []string, valueFiles []string) error {
	args := []string{}
	args = append(args, "install", "--name", releaseName, "--namespace", ns, chart)
	if timeout != nil {
		args = append(args, "--timeout", strconv.Itoa(*timeout))
	}
	if version != nil {
		args = append(args, "--version", *version)
	}
	for _, value := range values {
		args = append(args, "--set", value)
	}
	for _, valueFile := range valueFiles {
		args = append(args, "--values", valueFile)
	}
	return h.runHelm(args...)
}

// UpgradeChart upgrades a helm chart according with given helm flags
func (h *HelmCLI) UpgradeChart(chart string, releaseName string, ns string, version *string, install bool,
	timeout *int, force bool, wait bool, values []string, valueFiles []string) error {
	args := []string{}
	args = append(args, "upgrade")
	args = append(args, "--namespace", ns)
	if install {
		args = append(args, "--install")
	}
	if wait {
		args = append(args, "--wait")
	}
	if force {
		args = append(args, "--force")
	}
	if timeout != nil {
		args = append(args, "--timeout", strconv.Itoa(*timeout))
	}
	if version != nil {
		args = append(args, "--version", *version)
	}
	for _, value := range values {
		args = append(args, "--set", value)
	}
	for _, valueFile := range valueFiles {
		args = append(args, "--values", valueFile)
	}
	args = append(args, releaseName, chart)
	return h.runHelm(args...)
}

// DeleteRelease removes the given release
func (h *HelmCLI) DeleteRelease(releaseName string, purge bool) error {
	args := []string{}
	args = append(args, "delete")
	if purge {
		args = append(args, "--purge")
	}
	args = append(args, releaseName)
	return h.runHelm(args...)
}

// ListCharts execute the helm list command and returns its output
func (h *HelmCLI) ListCharts() (string, error) {
	return h.runHelmWithOutput("list")
}

// SearchChartVersions search all version of the given chart
func (h *HelmCLI) SearchChartVersions(chart string) ([]string, error) {
	output, err := h.runHelmWithOutput("search", chart, "--versions")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to search chart '%s'", chart)
	}
	versions := []string{}
	lines := strings.Split(output, "\n")
	for _, line := range lines[2:] {
		fields := strings.Fields(line)
		if len(fields) > 1 {
			v := fields[1]
			if v != "" {
				versions = append(versions, v)
			}
		}
	}
	return versions, nil
}

// FindChart find a chart in the current working directory, if no chart file is found an error is returned
func (h *HelmCLI) FindChart() (string, error) {
	dir := h.CWD
	chartFile := filepath.Join(dir, "Chart.yaml")
	exists, err := util.FileExists(chartFile)
	if err != nil {
		return "", errors.Wrapf(err, "no Chart.yaml file found in directory '%s'", dir)
	}
	if !exists {
		files, err := filepath.Glob("*/Chart.yaml")
		if err != nil {
			return "", errors.Wrap(err, "no Chart.yaml file found")
		}
		if len(files) > 0 {
			chartFile = files[0]
		} else {
			files, err = filepath.Glob("*/*/Chart.yaml")
			if err != nil {
				return "", errors.Wrap(err, "no Chart.yaml file found")
			}
			if len(files) > 0 {
				for _, file := range files {
					if !strings.HasSuffix(file, "/preview/Chart.yaml") {
						return file, nil
					}
				}
			}
		}
	}
	return chartFile, nil
}

// StatusRelease returns the output of the helm status command for a given release
func (h *HelmCLI) StatusRelease(releaseName string) error {
	return h.runHelm("status", releaseName)
}

// StatusReleases returns the status of all installed releases
func (h *HelmCLI) StatusReleases() (map[string]string, error) {
	output, err := h.ListCharts()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list the installed chart releases")
	}
	lines := strings.Split(output, "\n")
	statusMap := map[string]string{}
	for _, line := range lines[2:] {
		fields := strings.Split(line, "\t")
		if len(fields) > 3 {
			release := strings.TrimSpace(fields[0])
			status := strings.TrimSpace(fields[3])
			statusMap[release] = status
		}
	}
	return statusMap, nil
}

// Lint lints the helm chart from the current working directory and returns the warnings in the output
func (h *HelmCLI) Lint() (string, error) {
	return h.runHelmWithOutput("lint")
}

// Version executes the helm version command and returns its output
func (h *HelmCLI) Version(tls bool) (string, error) {
	args := []string{}
	args = append(args, "version", "--short")
	if tls {
		args = append(args, "--tls")
	}
	return h.runHelmWithOutput(args...)
}

// PackageChart packages the chart from the current working directory
func (h *HelmCLI) PackageChart() error {
	return h.runHelm("package", h.CWD)
}
