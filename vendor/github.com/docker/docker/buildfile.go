package docker

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/dotcloud/docker/utils"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"
)

type BuildFile interface {
	Build(io.Reader) (string, error)
	CmdFrom(string) error
	CmdRun(string) error
}

type buildFile struct {
	runtime *Runtime
	builder *Builder
	srv     *Server

	image      string
	maintainer string
	config     *Config
	context    string
	verbose    bool

	tmpContainers map[string]struct{}
	tmpImages     map[string]struct{}

	out io.Writer
}

func (b *buildFile) clearTmp(containers, images map[string]struct{}) {
	for c := range containers {
		tmp := b.runtime.Get(c)
		b.runtime.Destroy(tmp)
		utils.Debugf("Removing container %s", c)
	}
	for i := range images {
		b.runtime.graph.Delete(i)
		utils.Debugf("Removing image %s", i)
	}
}

func (b *buildFile) CmdFrom(name string) error {
	image, err := b.runtime.repositories.LookupImage(name)
	if err != nil {
		if b.runtime.graph.IsNotExist(err) {
			remote, tag := utils.ParseRepositoryTag(name)
			if err := b.srv.ImagePull(remote, tag, b.out, utils.NewStreamFormatter(false), nil); err != nil {
				return err
			}
			image, err = b.runtime.repositories.LookupImage(name)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	b.image = image.ID
	b.config = &Config{}
	if b.config.Env == nil || len(b.config.Env) == 0 {
		b.config.Env = append(b.config.Env, "HOME=/", "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")
	}
	return nil
}

func (b *buildFile) CmdMaintainer(name string) error {
	b.maintainer = name
	return b.commit("", b.config.Cmd, fmt.Sprintf("MAINTAINER %s", name))
}

func (b *buildFile) CmdRun(args string) error {
	if b.image == "" {
		return fmt.Errorf("Please provide a source image with `from` prior to run")
	}
	config, _, _, err := ParseRun([]string{b.image, "/bin/sh", "-c", args}, nil)
	if err != nil {
		return err
	}

	cmd := b.config.Cmd
	b.config.Cmd = nil
	MergeConfig(b.config, config)

	utils.Debugf("Command to be executed: %v", b.config.Cmd)

	if cache, err := b.srv.ImageGetCached(b.image, b.config); err != nil {
		return err
	} else if cache != nil {
		fmt.Fprintf(b.out, " ---> Using cache\n")
		utils.Debugf("[BUILDER] Use cached version")
		b.image = cache.ID
		return nil
	} else {
		utils.Debugf("[BUILDER] Cache miss")
	}

	cid, err := b.run()
	if err != nil {
		return err
	}
	if err := b.commit(cid, cmd, "run"); err != nil {
		return err
	}
	b.config.Cmd = cmd
	return nil
}

func (b *buildFile) FindEnvKey(key string) int {
	for k, envVar := range b.config.Env {
		envParts := strings.SplitN(envVar, "=", 2)
		if key == envParts[0] {
			return k
		}
	}
	return -1
}

func (b *buildFile) ReplaceEnvMatches(value string) (string, error) {
	exp, err := regexp.Compile("(\\\\\\\\+|[^\\\\]|\\b|\\A)\\$({?)([[:alnum:]_]+)(}?)")
	if err != nil {
		return value, err
	}
	matches := exp.FindAllString(value, -1)
	for _, match := range matches {
		match = match[strings.Index(match, "$"):]
		matchKey := strings.Trim(match, "${}")

		for _, envVar := range b.config.Env {
			envParts := strings.SplitN(envVar, "=", 2)
			envKey := envParts[0]
			envValue := envParts[1]

			if envKey == matchKey {
				value = strings.Replace(value, match, envValue, -1)
				break
			}
		}
	}
	return value, nil
}

func (b *buildFile) CmdEnv(args string) error {
	tmp := strings.SplitN(args, " ", 2)
	if len(tmp) != 2 {
		return fmt.Errorf("Invalid ENV format")
	}
	key := strings.Trim(tmp[0], " \t")
	value := strings.Trim(tmp[1], " \t")

	envKey := b.FindEnvKey(key)
	replacedValue, err := b.ReplaceEnvMatches(value)
	if err != nil {
		return err
	}
	replacedVar := fmt.Sprintf("%s=%s", key, replacedValue)

	if envKey >= 0 {
		b.config.Env[envKey] = replacedVar
	} else {
		b.config.Env = append(b.config.Env, replacedVar)
	}
	return b.commit("", b.config.Cmd, fmt.Sprintf("ENV %s", replacedVar))
}

func (b *buildFile) CmdCmd(args string) error {
	var cmd []string
	if err := json.Unmarshal([]byte(args), &cmd); err != nil {
		utils.Debugf("Error unmarshalling: %s, setting cmd to /bin/sh -c", err)
		cmd = []string{"/bin/sh", "-c", args}
	}
	if err := b.commit("", cmd, fmt.Sprintf("CMD %v", cmd)); err != nil {
		return err
	}
	b.config.Cmd = cmd
	return nil
}

func (b *buildFile) CmdExpose(args string) error {
	ports := strings.Split(args, " ")
	b.config.PortSpecs = append(ports, b.config.PortSpecs...)
	return b.commit("", b.config.Cmd, fmt.Sprintf("EXPOSE %v", ports))
}

func (b *buildFile) CmdInsert(args string) error {
	return fmt.Errorf("INSERT has been deprecated. Please use ADD instead")
}

func (b *buildFile) CmdCopy(args string) error {
	return fmt.Errorf("COPY has been deprecated. Please use ADD instead")
}

func (b *buildFile) CmdEntrypoint(args string) error {
	if args == "" {
		return fmt.Errorf("Entrypoint cannot be empty")
	}

	var entrypoint []string
	if err := json.Unmarshal([]byte(args), &entrypoint); err != nil {
		b.config.Entrypoint = []string{"/bin/sh", "-c", args}
	} else {
		b.config.Entrypoint = entrypoint
	}
	if err := b.commit("", b.config.Cmd, fmt.Sprintf("ENTRYPOINT %s", args)); err != nil {
		return err
	}
	return nil
}

func (b *buildFile) CmdVolume(args string) error {
	if args == "" {
		return fmt.Errorf("Volume cannot be empty")
	}

	var volume []string
	if err := json.Unmarshal([]byte(args), &volume); err != nil {
		volume = []string{args}
	}
	if b.config.Volumes == nil {
		b.config.Volumes = NewPathOpts()
	}
	for _, v := range volume {
		b.config.Volumes[v] = struct{}{}
	}
	if err := b.commit("", b.config.Cmd, fmt.Sprintf("VOLUME %s", args)); err != nil {
		return err
	}
	return nil
}

func (b *buildFile) addRemote(container *Container, orig, dest string) error {
	file, err := utils.Download(orig, ioutil.Discard)
	if err != nil {
		return err
	}
	defer file.Body.Close()

	// If the destination is a directory, figure out the filename.
	if strings.HasSuffix(dest, "/") {
		u, err := url.Parse(orig)
		if err != nil {
			return err
		}
		path := u.Path
		if strings.HasSuffix(path, "/") {
			path = path[:len(path)-1]
		}
		parts := strings.Split(path, "/")
		filename := parts[len(parts)-1]
		if filename == "" {
			return fmt.Errorf("cannot determine filename from url: %s", u)
		}
		dest = dest + filename
	}

	return container.Inject(file.Body, dest)
}

func (b *buildFile) addContext(container *Container, orig, dest string) error {
	origPath := path.Join(b.context, orig)
	destPath := path.Join(container.RootfsPath(), dest)
	// Preserve the trailing '/'
	if strings.HasSuffix(dest, "/") {
		destPath = destPath + "/"
	}
	if !strings.HasPrefix(origPath, b.context) {
		return fmt.Errorf("Forbidden path: %s", origPath)
	}
	fi, err := os.Stat(origPath)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		if err := CopyWithTar(origPath, destPath); err != nil {
			return err
		}
		// First try to unpack the source as an archive
	} else if err := UntarPath(origPath, destPath); err != nil {
		utils.Debugf("Couldn't untar %s to %s: %s", origPath, destPath, err)
		// If that fails, just copy it as a regular file
		if err := os.MkdirAll(path.Dir(destPath), 0755); err != nil {
			return err
		}
		if err := CopyWithTar(origPath, destPath); err != nil {
			return err
		}
	}
	return nil
}

func (b *buildFile) CmdAdd(args string) error {
	if b.context == "" {
		return fmt.Errorf("No context given. Impossible to use ADD")
	}
	tmp := strings.SplitN(args, " ", 2)
	if len(tmp) != 2 {
		return fmt.Errorf("Invalid ADD format")
	}

	orig, err := b.ReplaceEnvMatches(strings.Trim(tmp[0], " \t"))
	if err != nil {
		return err
	}

	dest, err := b.ReplaceEnvMatches(strings.Trim(tmp[1], " \t"))
	if err != nil {
		return err
	}

	cmd := b.config.Cmd
	b.config.Cmd = []string{"/bin/sh", "-c", fmt.Sprintf("#(nop) ADD %s in %s", orig, dest)}

	b.config.Image = b.image
	// Create the container and start it
	container, err := b.builder.Create(b.config)
	if err != nil {
		return err
	}
	b.tmpContainers[container.ID] = struct{}{}

	if err := container.EnsureMounted(); err != nil {
		return err
	}
	defer container.Unmount()

	if utils.IsURL(orig) {
		if err := b.addRemote(container, orig, dest); err != nil {
			return err
		}
	} else {
		if err := b.addContext(container, orig, dest); err != nil {
			return err
		}
	}

	if err := b.commit(container.ID, cmd, fmt.Sprintf("ADD %s in %s", orig, dest)); err != nil {
		return err
	}
	b.config.Cmd = cmd
	return nil
}

func (b *buildFile) run() (string, error) {
	if b.image == "" {
		return "", fmt.Errorf("Please provide a source image with `from` prior to run")
	}
	b.config.Image = b.image

	// Create the container and start it
	c, err := b.builder.Create(b.config)
	if err != nil {
		return "", err
	}
	b.tmpContainers[c.ID] = struct{}{}
	fmt.Fprintf(b.out, " ---> Running in %s\n", utils.TruncateID(c.ID))

	// override the entry point that may have been picked up from the base image
	c.Path = b.config.Cmd[0]
	c.Args = b.config.Cmd[1:]

	//start the container
	hostConfig := &HostConfig{}
	if err := c.Start(hostConfig); err != nil {
		return "", err
	}

	if b.verbose {
		err = <-c.Attach(nil, nil, b.out, b.out)
		if err != nil {
			return "", err
		}
	}

	// Wait for it to finish
	if ret := c.Wait(); ret != 0 {
		return "", fmt.Errorf("The command %v returned a non-zero code: %d", b.config.Cmd, ret)
	}

	return c.ID, nil
}

// Commit the container <id> with the autorun command <autoCmd>
func (b *buildFile) commit(id string, autoCmd []string, comment string) error {
	if b.image == "" {
		return fmt.Errorf("Please provide a source image with `from` prior to commit")
	}
	b.config.Image = b.image
	if id == "" {
		cmd := b.config.Cmd
		b.config.Cmd = []string{"/bin/sh", "-c", "#(nop) " + comment}
		defer func(cmd []string) { b.config.Cmd = cmd }(cmd)

		if cache, err := b.srv.ImageGetCached(b.image, b.config); err != nil {
			return err
		} else if cache != nil {
			fmt.Fprintf(b.out, " ---> Using cache\n")
			utils.Debugf("[BUILDER] Use cached version")
			b.image = cache.ID
			return nil
		} else {
			utils.Debugf("[BUILDER] Cache miss")
		}
		container, err := b.builder.Create(b.config)
		if err != nil {
			return err
		}
		b.tmpContainers[container.ID] = struct{}{}
		fmt.Fprintf(b.out, " ---> Running in %s\n", utils.TruncateID(container.ID))
		id = container.ID
		if err := container.EnsureMounted(); err != nil {
			return err
		}
		defer container.Unmount()
	}

	container := b.runtime.Get(id)
	if container == nil {
		return fmt.Errorf("An error occured while creating the container")
	}

	// Note: Actually copy the struct
	autoConfig := *b.config
	autoConfig.Cmd = autoCmd
	// Commit the container
	image, err := b.builder.Commit(container, "", "", "", b.maintainer, &autoConfig)
	if err != nil {
		return err
	}
	b.tmpImages[image.ID] = struct{}{}
	b.image = image.ID
	return nil
}

func (b *buildFile) Build(context io.Reader) (string, error) {
	// FIXME: @creack any reason for using /tmp instead of ""?
	// FIXME: @creack "name" is a terrible variable name
	name, err := ioutil.TempDir("/tmp", "docker-build")
	if err != nil {
		return "", err
	}
	if err := Untar(context, name); err != nil {
		return "", err
	}
	defer os.RemoveAll(name)
	b.context = name
	dockerfile, err := os.Open(path.Join(name, "Dockerfile"))
	if err != nil {
		return "", fmt.Errorf("Can't build a directory with no Dockerfile")
	}
	// FIXME: "file" is also a terrible variable name ;)
	file := bufio.NewReader(dockerfile)
	stepN := 0
	for {
		line, err := file.ReadString('\n')
		if err != nil {
			if err == io.EOF && line == "" {
				break
			} else if err != io.EOF {
				return "", err
			}
		}
		line = strings.Trim(strings.Replace(line, "\t", " ", -1), " \t\r\n")
		// Skip comments and empty line
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		tmp := strings.SplitN(line, " ", 2)
		if len(tmp) != 2 {
			return "", fmt.Errorf("Invalid Dockerfile format")
		}
		instruction := strings.ToLower(strings.Trim(tmp[0], " "))
		arguments := strings.Trim(tmp[1], " ")
		stepN += 1
		// FIXME: only count known instructions as build steps
		fmt.Fprintf(b.out, "Step %d : %s %s\n", stepN, strings.ToUpper(instruction), arguments)

		method, exists := reflect.TypeOf(b).MethodByName("Cmd" + strings.ToUpper(instruction[:1]) + strings.ToLower(instruction[1:]))
		if !exists {
			fmt.Fprintf(b.out, "# Skipping unknown instruction %s\n", strings.ToUpper(instruction))
			continue
		}
		ret := method.Func.Call([]reflect.Value{reflect.ValueOf(b), reflect.ValueOf(arguments)})[0].Interface()
		if ret != nil {
			return "", ret.(error)
		}

		fmt.Fprintf(b.out, " ---> %v\n", utils.TruncateID(b.image))
	}
	if b.image != "" {
		fmt.Fprintf(b.out, "Successfully built %s\n", utils.TruncateID(b.image))
		return b.image, nil
	}
	return "", fmt.Errorf("An error occured during the build\n")
}

func NewBuildFile(srv *Server, out io.Writer, verbose bool) BuildFile {
	return &buildFile{
		builder:       NewBuilder(srv.runtime),
		runtime:       srv.runtime,
		srv:           srv,
		config:        &Config{},
		out:           out,
		tmpContainers: make(map[string]struct{}),
		tmpImages:     make(map[string]struct{}),
		verbose:       verbose,
	}
}
