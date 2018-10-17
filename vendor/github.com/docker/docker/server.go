package docker

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dotcloud/docker/auth"
	"github.com/dotcloud/docker/registry"
	"github.com/dotcloud/docker/utils"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"
)

func (srv *Server) DockerVersion() APIVersion {
	return APIVersion{
		Version:   VERSION,
		GitCommit: GITCOMMIT,
		GoVersion: runtime.Version(),
	}
}

// simpleVersionInfo is a simple implementation of
// the interface VersionInfo, which is used
// to provide version information for some product,
// component, etc. It stores the product name and the version
// in string and returns them on calls to Name() and Version().
type simpleVersionInfo struct {
	name    string
	version string
}

func (v *simpleVersionInfo) Name() string {
	return v.name
}

func (v *simpleVersionInfo) Version() string {
	return v.version
}

// versionCheckers() returns version informations of:
// docker, go, git-commit (of the docker) and the host's kernel.
//
// Such information will be used on call to NewRegistry().
func (srv *Server) versionInfos() []registry.VersionInfo {
	v := srv.DockerVersion()
	ret := make([]registry.VersionInfo, 0, 4)
	ret = append(ret, &simpleVersionInfo{"docker", v.Version})

	if len(v.GoVersion) > 0 {
		ret = append(ret, &simpleVersionInfo{"go", v.GoVersion})
	}
	if len(v.GitCommit) > 0 {
		ret = append(ret, &simpleVersionInfo{"git-commit", v.GitCommit})
	}
	kernelVersion, err := utils.GetKernelVersion()
	if err == nil {
		ret = append(ret, &simpleVersionInfo{"kernel", kernelVersion.String()})
	}

	return ret
}

func (srv *Server) ContainerKill(name string) error {
	if container := srv.runtime.Get(name); container != nil {
		if err := container.Kill(); err != nil {
			return fmt.Errorf("Error killing container %s: %s", name, err)
		}
		srv.LogEvent("kill", name)
	} else {
		return fmt.Errorf("No such container: %s", name)
	}
	return nil
}

func (srv *Server) ContainerExport(name string, out io.Writer) error {
	if container := srv.runtime.Get(name); container != nil {

		data, err := container.Export()
		if err != nil {
			return err
		}

		// Stream the entire contents of the container (basically a volatile snapshot)
		if _, err := io.Copy(out, data); err != nil {
			return err
		}
		srv.LogEvent("export", name)
		return nil
	}
	return fmt.Errorf("No such container: %s", name)
}

func (srv *Server) ImagesSearch(term string) ([]APISearch, error) {
	r, err := registry.NewRegistry(srv.runtime.root, nil, srv.versionInfos()...)
	if err != nil {
		return nil, err
	}
	results, err := r.SearchRepositories(term)
	if err != nil {
		return nil, err
	}

	var outs []APISearch
	for _, repo := range results.Results {
		var out APISearch
		out.Description = repo["description"]
		out.Name = repo["name"]
		outs = append(outs, out)
	}
	return outs, nil
}

func (srv *Server) ImageInsert(name, url, path string, out io.Writer, sf *utils.StreamFormatter) (string, error) {
	out = utils.NewWriteFlusher(out)
	img, err := srv.runtime.repositories.LookupImage(name)
	if err != nil {
		return "", err
	}

	file, err := utils.Download(url, out)
	if err != nil {
		return "", err
	}
	defer file.Body.Close()

	config, _, _, err := ParseRun([]string{img.ID, "echo", "insert", url, path}, srv.runtime.capabilities)
	if err != nil {
		return "", err
	}

	b := NewBuilder(srv.runtime)
	c, err := b.Create(config)
	if err != nil {
		return "", err
	}

	if err := c.Inject(utils.ProgressReader(file.Body, int(file.ContentLength), out, sf.FormatProgress("Downloading", "%8v/%v (%v)"), sf), path); err != nil {
		return "", err
	}
	// FIXME: Handle custom repo, tag comment, author
	img, err = b.Commit(c, "", "", img.Comment, img.Author, nil)
	if err != nil {
		return "", err
	}
	out.Write(sf.FormatStatus(img.ID))
	return img.ShortID(), nil
}

func (srv *Server) ImagesViz(out io.Writer) error {
	images, _ := srv.runtime.graph.All()
	if images == nil {
		return nil
	}
	out.Write([]byte("digraph docker {\n"))

	var (
		parentImage *Image
		err         error
	)
	for _, image := range images {
		parentImage, err = image.GetParent()
		if err != nil {
			return fmt.Errorf("Error while getting parent image: %v", err)
		}
		if parentImage != nil {
			out.Write([]byte(" \"" + parentImage.ShortID() + "\" -> \"" + image.ShortID() + "\"\n"))
		} else {
			out.Write([]byte(" base -> \"" + image.ShortID() + "\" [style=invis]\n"))
		}
	}

	reporefs := make(map[string][]string)

	for name, repository := range srv.runtime.repositories.Repositories {
		for tag, id := range repository {
			reporefs[utils.TruncateID(id)] = append(reporefs[utils.TruncateID(id)], fmt.Sprintf("%s:%s", name, tag))
		}
	}

	for id, repos := range reporefs {
		out.Write([]byte(" \"" + id + "\" [label=\"" + id + "\\n" + strings.Join(repos, "\\n") + "\",shape=box,fillcolor=\"paleturquoise\",style=\"filled,rounded\"];\n"))
	}
	out.Write([]byte(" base [style=invisible]\n}\n"))
	return nil
}

func (srv *Server) Images(all bool, filter string) ([]APIImages, error) {
	var (
		allImages map[string]*Image
		err       error
	)
	if all {
		allImages, err = srv.runtime.graph.Map()
	} else {
		allImages, err = srv.runtime.graph.Heads()
	}
	if err != nil {
		return nil, err
	}
	outs := []APIImages{} //produce [] when empty instead of 'null'
	for name, repository := range srv.runtime.repositories.Repositories {
		if filter != "" && name != filter {
			continue
		}
		for tag, id := range repository {
			var out APIImages
			image, err := srv.runtime.graph.Get(id)
			if err != nil {
				log.Printf("Warning: couldn't load %s from %s/%s: %s", id, name, tag, err)
				continue
			}
			delete(allImages, id)
			out.Repository = name
			out.Tag = tag
			out.ID = image.ID
			out.Created = image.Created.Unix()
			out.Size = image.Size
			out.VirtualSize = image.getParentsSize(0) + image.Size
			outs = append(outs, out)
		}
	}
	// Display images which aren't part of a
	if filter == "" {
		for _, image := range allImages {
			var out APIImages
			out.ID = image.ID
			out.Created = image.Created.Unix()
			out.Size = image.Size
			out.VirtualSize = image.getParentsSize(0) + image.Size
			outs = append(outs, out)
		}
	}
	return outs, nil
}

func (srv *Server) DockerInfo() *APIInfo {
	images, _ := srv.runtime.graph.All()
	var imgcount int
	if images == nil {
		imgcount = 0
	} else {
		imgcount = len(images)
	}
	lxcVersion := ""
	if output, err := exec.Command("lxc-version").CombinedOutput(); err == nil {
		outputStr := string(output)
		if len(strings.SplitN(outputStr, ":", 2)) == 2 {
			lxcVersion = strings.TrimSpace(strings.SplitN(string(output), ":", 2)[1])
		}
	}
	kernelVersion := "<unknown>"
	if kv, err := utils.GetKernelVersion(); err == nil {
		kernelVersion = kv.String()
	}

	return &APIInfo{
		Containers:      len(srv.runtime.List()),
		Images:          imgcount,
		MemoryLimit:     srv.runtime.capabilities.MemoryLimit,
		SwapLimit:       srv.runtime.capabilities.SwapLimit,
		Debug:           os.Getenv("DEBUG") != "",
		NFd:             utils.GetTotalUsedFds(),
		NGoroutines:     runtime.NumGoroutine(),
		LXCVersion:      lxcVersion,
		NEventsListener: len(srv.events),
		KernelVersion:   kernelVersion,
	}
}

func (srv *Server) ImageHistory(name string) ([]APIHistory, error) {
	image, err := srv.runtime.repositories.LookupImage(name)
	if err != nil {
		return nil, err
	}

	lookupMap := make(map[string][]string)
	for name, repository := range srv.runtime.repositories.Repositories {
		for tag, id := range repository {
			// If the ID already has a reverse lookup, do not update it unless for "latest"
			if _, exists := lookupMap[id]; !exists {
				lookupMap[id] = []string{}
			}
			lookupMap[id] = append(lookupMap[id], name+":"+tag)
		}
	}

	outs := []APIHistory{} //produce [] when empty instead of 'null'
	err = image.WalkHistory(func(img *Image) error {
		var out APIHistory
		out.ID = srv.runtime.repositories.ImageName(img.ShortID())
		out.Created = img.Created.Unix()
		out.CreatedBy = strings.Join(img.ContainerConfig.Cmd, " ")
		out.Tags = lookupMap[img.ID]
		outs = append(outs, out)
		return nil
	})
	return outs, nil

}

func (srv *Server) ContainerTop(name, ps_args string) (*APITop, error) {
	if container := srv.runtime.Get(name); container != nil {
		output, err := exec.Command("lxc-ps", "--name", container.ID, "--", ps_args).CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("Error trying to use lxc-ps: %s (%s)", err, output)
		}
		procs := APITop{}
		for i, line := range strings.Split(string(output), "\n") {
			if len(line) == 0 {
				continue
			}
			words := []string{}
			scanner := bufio.NewScanner(strings.NewReader(line))
			scanner.Split(bufio.ScanWords)
			if !scanner.Scan() {
				return nil, fmt.Errorf("Error trying to use lxc-ps")
			}
			// no scanner.Text because we skip container id
			for scanner.Scan() {
				words = append(words, scanner.Text())
			}
			if i == 0 {
				procs.Titles = words
			} else {
				procs.Processes = append(procs.Processes, words)
			}
		}
		return &procs, nil

	}
	return nil, fmt.Errorf("No such container: %s", name)
}

func (srv *Server) ContainerChanges(name string) ([]Change, error) {
	if container := srv.runtime.Get(name); container != nil {
		return container.Changes()
	}
	return nil, fmt.Errorf("No such container: %s", name)
}

func (srv *Server) Containers(all, size bool, n int, since, before string) []APIContainers {
	var foundBefore bool
	var displayed int
	retContainers := []APIContainers{}

	for _, container := range srv.runtime.List() {
		if !container.State.Running && !all && n == -1 && since == "" && before == "" {
			continue
		}
		if before != "" {
			if container.ShortID() == before {
				foundBefore = true
				continue
			}
			if !foundBefore {
				continue
			}
		}
		if displayed == n {
			break
		}
		if container.ShortID() == since {
			break
		}
		displayed++

		c := APIContainers{
			ID: container.ID,
		}
		c.Image = srv.runtime.repositories.ImageName(container.Image)
		c.Command = fmt.Sprintf("%s %s", container.Path, strings.Join(container.Args, " "))
		c.Created = container.Created.Unix()
		c.Status = container.State.String()
		c.Ports = container.NetworkSettings.PortMappingHuman()
		if size {
			c.SizeRw, c.SizeRootFs = container.GetSize()
		}
		retContainers = append(retContainers, c)
	}
	return retContainers
}

func (srv *Server) ContainerCommit(name, repo, tag, author, comment string, config *Config) (string, error) {
	container := srv.runtime.Get(name)
	if container == nil {
		return "", fmt.Errorf("No such container: %s", name)
	}
	img, err := NewBuilder(srv.runtime).Commit(container, repo, tag, comment, author, config)
	if err != nil {
		return "", err
	}
	return img.ShortID(), err
}

func (srv *Server) ContainerTag(name, repo, tag string, force bool) error {
	if err := srv.runtime.repositories.Set(repo, tag, name, force); err != nil {
		return err
	}
	return nil
}

func (srv *Server) pullImage(r *registry.Registry, out io.Writer, imgID, endpoint string, token []string, sf *utils.StreamFormatter) error {
	history, err := r.GetRemoteHistory(imgID, endpoint, token)
	if err != nil {
		return err
	}

	// FIXME: Try to stream the images?
	// FIXME: Launch the getRemoteImage() in goroutines
	for _, id := range history {
		if !srv.runtime.graph.Exists(id) {
			out.Write(sf.FormatStatus("Pulling %s metadata", id))
			imgJSON, imgSize, err := r.GetRemoteImageJSON(id, endpoint, token)
			if err != nil {
				// FIXME: Keep goging in case of error?
				return err
			}
			img, err := NewImgJSON(imgJSON)
			if err != nil {
				return fmt.Errorf("Failed to parse json: %s", err)
			}

			// Get the layer
			out.Write(sf.FormatStatus("Pulling %s fs layer", id))
			layer, err := r.GetRemoteImageLayer(img.ID, endpoint, token)
			if err != nil {
				return err
			}
			defer layer.Close()
			if err := srv.runtime.graph.Register(utils.ProgressReader(layer, imgSize, out, sf.FormatProgress("Downloading", "%8v/%v (%v)"), sf), false, img); err != nil {
				return err
			}
		}
	}
	return nil
}

func (srv *Server) pullRepository(r *registry.Registry, out io.Writer, localName, remoteName, askedTag, indexEp string, sf *utils.StreamFormatter) error {
	out.Write(sf.FormatStatus("Pulling repository %s", localName))

	repoData, err := r.GetRepositoryData(indexEp, remoteName)
	if err != nil {
		return err
	}

	utils.Debugf("Updating checksums")
	// Reload the json file to make sure not to overwrite faster sums
	if err := srv.runtime.graph.UpdateChecksums(repoData.ImgList); err != nil {
		return err
	}

	utils.Debugf("Retrieving the tag list")
	tagsList, err := r.GetRemoteTags(repoData.Endpoints, remoteName, repoData.Tokens)
	if err != nil {
		utils.Debugf("%v", err)
		return err
	}

	for tag, id := range tagsList {
		repoData.ImgList[id] = &registry.ImgData{
			ID:       id,
			Tag:      tag,
			Checksum: "",
		}
	}

	utils.Debugf("Registering tags")
	// If no tag has been specified, pull them all
	if askedTag == "" {
		for tag, id := range tagsList {
			repoData.ImgList[id].Tag = tag
		}
	} else {
		// Otherwise, check that the tag exists and use only that one
		id, exists := tagsList[askedTag]
		if !exists {
			return fmt.Errorf("Tag %s not found in repository %s", askedTag, localName)
		}
		repoData.ImgList[id].Tag = askedTag
	}

	for _, img := range repoData.ImgList {
		if askedTag != "" && img.Tag != askedTag {
			utils.Debugf("(%s) does not match %s (id: %s), skipping", img.Tag, askedTag, img.ID)
			continue
		}

		if img.Tag == "" {
			utils.Debugf("Image (id: %s) present in this repository but untagged, skipping", img.ID)
			continue
		}
		out.Write(sf.FormatStatus("Pulling image %s (%s) from %s", img.ID, img.Tag, localName))
		success := false
		for _, ep := range repoData.Endpoints {
			if err := srv.pullImage(r, out, img.ID, ep, repoData.Tokens, sf); err != nil {
				out.Write(sf.FormatStatus("Error while retrieving image for tag: %s (%s); checking next endpoint", askedTag, err))
				continue
			}
			success = true
			break
		}
		if !success {
			return fmt.Errorf("Could not find repository on any of the indexed registries.")
		}
	}
	for tag, id := range tagsList {
		if askedTag != "" && tag != askedTag {
			continue
		}
		if err := srv.runtime.repositories.Set(localName, tag, id, true); err != nil {
			return err
		}
	}
	if err := srv.runtime.repositories.Save(); err != nil {
		return err
	}

	return nil
}

func (srv *Server) poolAdd(kind, key string) error {
	srv.Lock()
	defer srv.Unlock()

	if _, exists := srv.pullingPool[key]; exists {
		return fmt.Errorf("%s %s is already in progress", key, kind)
	}

	switch kind {
	case "pull":
		srv.pullingPool[key] = struct{}{}
		break
	case "push":
		srv.pushingPool[key] = struct{}{}
		break
	default:
		return fmt.Errorf("Unkown pool type")
	}
	return nil
}

func (srv *Server) poolRemove(kind, key string) error {
	switch kind {
	case "pull":
		delete(srv.pullingPool, key)
		break
	case "push":
		delete(srv.pushingPool, key)
		break
	default:
		return fmt.Errorf("Unkown pool type")
	}
	return nil
}

func (srv *Server) ImagePull(localName string, tag string, out io.Writer, sf *utils.StreamFormatter, authConfig *auth.AuthConfig) error {
	r, err := registry.NewRegistry(srv.runtime.root, authConfig, srv.versionInfos()...)
	if err != nil {
		return err
	}
	if err := srv.poolAdd("pull", localName+":"+tag); err != nil {
		return err
	}
	defer srv.poolRemove("pull", localName+":"+tag)

	// Resolve the Repository name from fqn to endpoint + name
	endpoint, remoteName, err := registry.ResolveRepositoryName(localName)
	if err != nil {
		return err
	}

	if endpoint == auth.IndexServerAddress() {
		// If pull "index.docker.io/foo/bar", it's stored locally under "foo/bar"
		localName = remoteName
	}

	out = utils.NewWriteFlusher(out)
	err = srv.pullRepository(r, out, localName, remoteName, tag, endpoint, sf)
	if err != nil {
		if err := srv.pullImage(r, out, remoteName, endpoint, nil, sf); err != nil {
			return err
		}
		return nil
	}

	return nil
}

// Retrieve the checksum of an image
// Priority:
// - Check on the stored checksums
// - Check if the archive exists, if it does not, ask the registry
// - If the archive does exists, process the checksum from it
// - If the archive does not exists and not found on registry, process checksum from layer
func (srv *Server) getChecksum(imageID string) (string, error) {
	// FIXME: Use in-memory map instead of reading the file each time
	if sums, err := srv.runtime.graph.getStoredChecksums(); err != nil {
		return "", err
	} else if checksum, exists := sums[imageID]; exists {
		return checksum, nil
	}

	img, err := srv.runtime.graph.Get(imageID)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(layerArchivePath(srv.runtime.graph.imageRoot(imageID))); err != nil {
		if os.IsNotExist(err) {
			// TODO: Ask the registry for the checksum
			//       As the archive is not there, it is supposed to come from a pull.
		} else {
			return "", err
		}
	}

	checksum, err := img.Checksum()
	if err != nil {
		return "", err
	}
	return checksum, nil
}

// Retrieve the all the images to be uploaded in the correct order
// Note: we can't use a map as it is not ordered
func (srv *Server) getImageList(localRepo map[string]string) ([]*registry.ImgData, error) {
	var imgList []*registry.ImgData

	imageSet := make(map[string]struct{})
	for tag, id := range localRepo {
		img, err := srv.runtime.graph.Get(id)
		if err != nil {
			return nil, err
		}
		img.WalkHistory(func(img *Image) error {
			if _, exists := imageSet[img.ID]; exists {
				return nil
			}
			imageSet[img.ID] = struct{}{}
			checksum, err := srv.getChecksum(img.ID)
			if err != nil {
				return err
			}
			imgList = append([]*registry.ImgData{{
				ID:       img.ID,
				Checksum: checksum,
				Tag:      tag,
			}}, imgList...)
			return nil
		})
	}
	return imgList, nil
}

func (srv *Server) pushRepository(r *registry.Registry, out io.Writer, localName, remoteName string, localRepo map[string]string, indexEp string, sf *utils.StreamFormatter) error {
	out = utils.NewWriteFlusher(out)
	out.Write(sf.FormatStatus("Processing checksums"))
	imgList, err := srv.getImageList(localRepo)
	if err != nil {
		return err
	}
	out.Write(sf.FormatStatus("Sending image list"))

	var repoData *registry.RepositoryData
	repoData, err = r.PushImageJSONIndex(indexEp, remoteName, imgList, false, nil)
	if err != nil {
		return err
	}

	for _, ep := range repoData.Endpoints {
		out.Write(sf.FormatStatus("Pushing repository %s (%d tags)", localName, len(localRepo)))
		// For each image within the repo, push them
		for _, elem := range imgList {
			if _, exists := repoData.ImgList[elem.ID]; exists {
				out.Write(sf.FormatStatus("Image %s already pushed, skipping", elem.ID))
				continue
			} else if r.LookupRemoteImage(elem.ID, ep, repoData.Tokens) {
				out.Write(sf.FormatStatus("Image %s already pushed, skipping", elem.ID))
				continue
			}
			if err := srv.pushImage(r, out, remoteName, elem.ID, ep, repoData.Tokens, sf); err != nil {
				// FIXME: Continue on error?
				return err
			}
			out.Write(sf.FormatStatus("Pushing tags for rev [%s] on {%s}", elem.ID, ep+"repositories/"+remoteName+"/tags/"+elem.Tag))
			if err := r.PushRegistryTag(remoteName, elem.ID, elem.Tag, ep, repoData.Tokens); err != nil {
				return err
			}
		}
	}

	if _, err := r.PushImageJSONIndex(indexEp, remoteName, imgList, true, repoData.Endpoints); err != nil {
		return err
	}

	return nil
}

func (srv *Server) pushImage(r *registry.Registry, out io.Writer, remote, imgID, ep string, token []string, sf *utils.StreamFormatter) error {
	out = utils.NewWriteFlusher(out)
	jsonRaw, err := ioutil.ReadFile(path.Join(srv.runtime.graph.Root, imgID, "json"))
	if err != nil {
		return fmt.Errorf("Error while retreiving the path for {%s}: %s", imgID, err)
	}
	out.Write(sf.FormatStatus("Pushing %s", imgID))

	// Make sure we have the image's checksum
	checksum, err := srv.getChecksum(imgID)
	if err != nil {
		return err
	}
	imgData := &registry.ImgData{
		ID:       imgID,
		Checksum: checksum,
	}

	// Send the json
	if err := r.PushImageJSONRegistry(imgData, jsonRaw, ep, token); err != nil {
		if err == registry.ErrAlreadyExists {
			out.Write(sf.FormatStatus("Image %s already pushed, skipping", imgData.ID))
			return nil
		}
		return err
	}

	// Retrieve the tarball to be sent
	var layerData *TempArchive
	// If the archive exists, use it
	file, err := os.Open(layerArchivePath(srv.runtime.graph.imageRoot(imgID)))
	if err != nil {
		if os.IsNotExist(err) {
			// If the archive does not exist, create one from the layer
			layerData, err = srv.runtime.graph.TempLayerArchive(imgID, Xz, sf, out)
			if err != nil {
				return fmt.Errorf("Failed to generate layer archive: %s", err)
			}
		} else {
			return err
		}
	} else {
		defer file.Close()
		st, err := file.Stat()
		if err != nil {
			return err
		}
		layerData = &TempArchive{
			File: file,
			Size: st.Size(),
		}
	}

	// Send the layer
	if err := r.PushImageLayerRegistry(imgData.ID, utils.ProgressReader(layerData, int(layerData.Size), out, sf.FormatProgress("Pushing", "%8v/%v (%v)"), sf), ep, token); err != nil {
		return err
	}
	return nil
}

// FIXME: Allow to interupt current push when new push of same image is done.
func (srv *Server) ImagePush(localName string, out io.Writer, sf *utils.StreamFormatter, authConfig *auth.AuthConfig) error {
	if err := srv.poolAdd("push", localName); err != nil {
		return err
	}
	defer srv.poolRemove("push", localName)

	// Resolve the Repository name from fqn to endpoint + name
	endpoint, remoteName, err := registry.ResolveRepositoryName(localName)
	if err != nil {
		return err
	}

	out = utils.NewWriteFlusher(out)
	img, err := srv.runtime.graph.Get(localName)
	r, err2 := registry.NewRegistry(srv.runtime.root, authConfig, srv.versionInfos()...)
	if err2 != nil {
		return err2
	}

	if err != nil {
		reposLen := len(srv.runtime.repositories.Repositories[localName])
		out.Write(sf.FormatStatus("The push refers to a repository [%s] (len: %d)", localName, reposLen))
		// If it fails, try to get the repository
		if localRepo, exists := srv.runtime.repositories.Repositories[localName]; exists {
			if err := srv.pushRepository(r, out, localName, remoteName, localRepo, endpoint, sf); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	var token []string
	out.Write(sf.FormatStatus("The push refers to an image: [%s]", localName))
	if err := srv.pushImage(r, out, remoteName, img.ID, endpoint, token, sf); err != nil {
		return err
	}
	return nil
}

func (srv *Server) ImageImport(src, repo, tag string, in io.Reader, out io.Writer, sf *utils.StreamFormatter) error {
	var archive io.Reader
	var resp *http.Response

	if src == "-" {
		archive = in
	} else {
		u, err := url.Parse(src)
		if err != nil {
			return err
		}
		if u.Scheme == "" {
			u.Scheme = "http"
			u.Host = src
			u.Path = ""
		}
		out.Write(sf.FormatStatus("Downloading from %s", u))
		// Download with curl (pretty progress bar)
		// If curl is not available, fallback to http.Get()
		resp, err = utils.Download(u.String(), out)
		if err != nil {
			return err
		}
		archive = utils.ProgressReader(resp.Body, int(resp.ContentLength), out, sf.FormatProgress("Importing", "%8v/%v (%v)"), sf)
	}
	img, err := srv.runtime.graph.Create(archive, nil, "Imported from "+src, "", nil)
	if err != nil {
		return err
	}
	// Optionally register the image at REPO/TAG
	if repo != "" {
		if err := srv.runtime.repositories.Set(repo, tag, img.ID, true); err != nil {
			return err
		}
	}
	out.Write(sf.FormatStatus(img.ShortID()))
	return nil
}

func (srv *Server) ContainerCreate(config *Config) (string, error) {

	if config.Memory != 0 && config.Memory < 524288 {
		return "", fmt.Errorf("Memory limit must be given in bytes (minimum 524288 bytes)")
	}

	if config.Memory > 0 && !srv.runtime.capabilities.MemoryLimit {
		config.Memory = 0
	}

	if config.Memory > 0 && !srv.runtime.capabilities.SwapLimit {
		config.MemorySwap = -1
	}
	b := NewBuilder(srv.runtime)
	container, err := b.Create(config)
	if err != nil {
		if srv.runtime.graph.IsNotExist(err) {
			return "", fmt.Errorf("No such image: %s", config.Image)
		}
		return "", err
	}
	srv.LogEvent("create", container.ShortID())
	return container.ShortID(), nil
}

func (srv *Server) ContainerRestart(name string, t int) error {
	if container := srv.runtime.Get(name); container != nil {
		if err := container.Restart(t); err != nil {
			return fmt.Errorf("Error restarting container %s: %s", name, err)
		}
		srv.LogEvent("restart", name)
	} else {
		return fmt.Errorf("No such container: %s", name)
	}
	return nil
}

func (srv *Server) ContainerDestroy(name string, removeVolume bool) error {
	if container := srv.runtime.Get(name); container != nil {
		if container.State.Running {
			return fmt.Errorf("Impossible to remove a running container, please stop it first")
		}
		volumes := make(map[string]struct{})
		// Store all the deleted containers volumes
		for _, volumeId := range container.Volumes {
			volumes[volumeId] = struct{}{}
		}
		if err := srv.runtime.Destroy(container); err != nil {
			return fmt.Errorf("Error destroying container %s: %s", name, err)
		}
		srv.LogEvent("destroy", name)

		if removeVolume {
			// Retrieve all volumes from all remaining containers
			usedVolumes := make(map[string]*Container)
			for _, container := range srv.runtime.List() {
				for _, containerVolumeId := range container.Volumes {
					usedVolumes[containerVolumeId] = container
				}
			}

			for volumeId := range volumes {
				// If the requested volu
				if c, exists := usedVolumes[volumeId]; exists {
					log.Printf("The volume %s is used by the container %s. Impossible to remove it. Skipping.\n", volumeId, c.ID)
					continue
				}
				if err := srv.runtime.volumes.Delete(volumeId); err != nil {
					return err
				}
			}
		}
	} else {
		return fmt.Errorf("No such container: %s", name)
	}
	return nil
}

var ErrImageReferenced = errors.New("Image referenced by a repository")

func (srv *Server) deleteImageAndChildren(id string, imgs *[]APIRmi) error {
	// If the image is referenced by a repo, do not delete
	if len(srv.runtime.repositories.ByID()[id]) != 0 {
		return ErrImageReferenced
	}
	// If the image is not referenced but has children, go recursive
	referenced := false
	byParents, err := srv.runtime.graph.ByParent()
	if err != nil {
		return err
	}
	for _, img := range byParents[id] {
		if err := srv.deleteImageAndChildren(img.ID, imgs); err != nil {
			if err != ErrImageReferenced {
				return err
			}
			referenced = true
		}
	}
	if referenced {
		return ErrImageReferenced
	}

	// If the image is not referenced and has no children, remove it
	byParents, err = srv.runtime.graph.ByParent()
	if err != nil {
		return err
	}
	if len(byParents[id]) == 0 {
		if err := srv.runtime.repositories.DeleteAll(id); err != nil {
			return err
		}
		err := srv.runtime.graph.Delete(id)
		if err != nil {
			return err
		}
		*imgs = append(*imgs, APIRmi{Deleted: utils.TruncateID(id)})
		srv.LogEvent("delete", utils.TruncateID(id))
		return nil
	}
	return nil
}

func (srv *Server) deleteImageParents(img *Image, imgs *[]APIRmi) error {
	if img.Parent != "" {
		parent, err := srv.runtime.graph.Get(img.Parent)
		if err != nil {
			return err
		}
		// Remove all children images
		if err := srv.deleteImageAndChildren(img.Parent, imgs); err != nil {
			return err
		}
		return srv.deleteImageParents(parent, imgs)
	}
	return nil
}

func (srv *Server) deleteImage(img *Image, repoName, tag string) ([]APIRmi, error) {
	imgs := []APIRmi{}

	//If delete by id, see if the id belong only to one repository
	if strings.Contains(img.ID, repoName) && tag == "" {
		for _, repoAndTag := range srv.runtime.repositories.ByID()[img.ID] {
			parsedRepo := strings.Split(repoAndTag, ":")[0]
			if strings.Contains(img.ID, repoName) {
				repoName = parsedRepo
				if len(srv.runtime.repositories.ByID()[img.ID]) == 1 && len(strings.Split(repoAndTag, ":")) > 1 {
					tag = strings.Split(repoAndTag, ":")[1]
				}
			} else if repoName != parsedRepo {
				// the id belongs to multiple repos, like base:latest and user:test,
				// in that case return conflict
				return imgs, nil
			}
		}
	}
	//Untag the current image
	tagDeleted, err := srv.runtime.repositories.Delete(repoName, tag)
	if err != nil {
		return nil, err
	}
	if tagDeleted {
		imgs = append(imgs, APIRmi{Untagged: img.ShortID()})
		srv.LogEvent("untag", img.ShortID())
	}
	if len(srv.runtime.repositories.ByID()[img.ID]) == 0 {
		if err := srv.deleteImageAndChildren(img.ID, &imgs); err != nil {
			if err != ErrImageReferenced {
				return imgs, err
			}
		} else if err := srv.deleteImageParents(img, &imgs); err != nil {
			if err != ErrImageReferenced {
				return imgs, err
			}
		}
	}
	return imgs, nil
}

func (srv *Server) ImageDelete(name string, autoPrune bool) ([]APIRmi, error) {
	img, err := srv.runtime.repositories.LookupImage(name)
	if err != nil {
		return nil, fmt.Errorf("No such image: %s", name)
	}
	if !autoPrune {
		if err := srv.runtime.graph.Delete(img.ID); err != nil {
			return nil, fmt.Errorf("Error deleting image %s: %s", name, err)
		}
		return nil, nil
	}

	var tag string
	if strings.Contains(name, ":") {
		nameParts := strings.Split(name, ":")
		name = nameParts[0]
		tag = nameParts[1]
	}

	return srv.deleteImage(img, name, tag)
}

func (srv *Server) ImageGetCached(imgID string, config *Config) (*Image, error) {

	// Retrieve all images
	images, err := srv.runtime.graph.All()
	if err != nil {
		return nil, err
	}

	// Store the tree in a map of map (map[parentId][childId])
	imageMap := make(map[string]map[string]struct{})
	for _, img := range images {
		if _, exists := imageMap[img.Parent]; !exists {
			imageMap[img.Parent] = make(map[string]struct{})
		}
		imageMap[img.Parent][img.ID] = struct{}{}
	}

	// Loop on the children of the given image and check the config
	for elem := range imageMap[imgID] {
		img, err := srv.runtime.graph.Get(elem)
		if err != nil {
			return nil, err
		}
		if CompareConfig(&img.ContainerConfig, config) {
			return img, nil
		}
	}
	return nil, nil
}

func (srv *Server) ContainerStart(name string, hostConfig *HostConfig) error {
	if container := srv.runtime.Get(name); container != nil {
		if err := container.Start(hostConfig); err != nil {
			return fmt.Errorf("Error starting container %s: %s", name, err)
		}
		srv.LogEvent("start", name)
	} else {
		return fmt.Errorf("No such container: %s", name)
	}
	return nil
}

func (srv *Server) ContainerStop(name string, t int) error {
	if container := srv.runtime.Get(name); container != nil {
		if err := container.Stop(t); err != nil {
			return fmt.Errorf("Error stopping container %s: %s", name, err)
		}
		srv.LogEvent("stop", name)
	} else {
		return fmt.Errorf("No such container: %s", name)
	}
	return nil
}

func (srv *Server) ContainerWait(name string) (int, error) {
	if container := srv.runtime.Get(name); container != nil {
		return container.Wait(), nil
	}
	return 0, fmt.Errorf("No such container: %s", name)
}

func (srv *Server) ContainerResize(name string, h, w int) error {
	if container := srv.runtime.Get(name); container != nil {
		return container.Resize(h, w)
	}
	return fmt.Errorf("No such container: %s", name)
}

func (srv *Server) ContainerAttach(name string, logs, stream, stdin, stdout, stderr bool, in io.ReadCloser, out io.Writer) error {
	container := srv.runtime.Get(name)
	if container == nil {
		return fmt.Errorf("No such container: %s", name)
	}
	//logs
	if logs {
		cLog, err := container.ReadLog("json")
		if err != nil && os.IsNotExist(err) {
			// Legacy logs
			utils.Debugf("Old logs format")
			if stdout {
				cLog, err := container.ReadLog("stdout")
				if err != nil {
					utils.Debugf("Error reading logs (stdout): %s", err)
				} else if _, err := io.Copy(out, cLog); err != nil {
					utils.Debugf("Error streaming logs (stdout): %s", err)
				}
			}
			if stderr {
				cLog, err := container.ReadLog("stderr")
				if err != nil {
					utils.Debugf("Error reading logs (stderr): %s", err)
				} else if _, err := io.Copy(out, cLog); err != nil {
					utils.Debugf("Error streaming logs (stderr): %s", err)
				}
			}
		} else if err != nil {
			utils.Debugf("Error reading logs (json): %s", err)
		} else {
			dec := json.NewDecoder(cLog)
			for {
				var l utils.JSONLog
				if err := dec.Decode(&l); err == io.EOF {
					break
				} else if err != nil {
					utils.Debugf("Error streaming logs: %s", err)
					break
				}
				if (l.Stream == "stdout" && stdout) || (l.Stream == "stderr" && stderr) {
					fmt.Fprintf(out, "%s", l.Log)
				}
			}
		}
	}

	//stream
	if stream {
		if container.State.Ghost {
			return fmt.Errorf("Impossible to attach to a ghost container")
		}

		var (
			cStdin           io.ReadCloser
			cStdout, cStderr io.Writer
			cStdinCloser     io.Closer
		)

		if stdin {
			r, w := io.Pipe()
			go func() {
				defer w.Close()
				defer utils.Debugf("Closing buffered stdin pipe")
				io.Copy(w, in)
			}()
			cStdin = r
			cStdinCloser = in
		}
		if stdout {
			cStdout = out
		}
		if stderr {
			cStderr = out
		}

		<-container.Attach(cStdin, cStdinCloser, cStdout, cStderr)

		// If we are in stdinonce mode, wait for the process to end
		// otherwise, simply return
		if container.Config.StdinOnce && !container.Config.Tty {
			container.Wait()
		}
	}
	return nil
}

func (srv *Server) ContainerInspect(name string) (*Container, error) {
	if container := srv.runtime.Get(name); container != nil {
		return container, nil
	}
	return nil, fmt.Errorf("No such container: %s", name)
}

func (srv *Server) ImageInspect(name string) (*Image, error) {
	if image, err := srv.runtime.repositories.LookupImage(name); err == nil && image != nil {
		return image, nil
	}
	return nil, fmt.Errorf("No such image: %s", name)
}

func NewServer(flGraphPath string, autoRestart, enableCors bool, dns ListOpts) (*Server, error) {
	if runtime.GOARCH != "amd64" {
		log.Fatalf("The docker runtime currently only supports amd64 (not %s). This will change in the future. Aborting.", runtime.GOARCH)
	}
	runtime, err := NewRuntime(flGraphPath, autoRestart, dns)
	if err != nil {
		return nil, err
	}
	srv := &Server{
		runtime:     runtime,
		enableCors:  enableCors,
		pullingPool: make(map[string]struct{}),
		pushingPool: make(map[string]struct{}),
		events:      make([]utils.JSONMessage, 0, 64), //only keeps the 64 last events
		listeners:   make(map[string]chan utils.JSONMessage),
	}
	runtime.srv = srv
	return srv, nil
}

func (srv *Server) LogEvent(action, id string) {
	now := time.Now().Unix()
	jm := utils.JSONMessage{Status: action, ID: id, Time: now}
	srv.events = append(srv.events, jm)
	for _, c := range srv.listeners {
		select { // non blocking channel
		case c <- jm:
		default:
		}
	}
}

type Server struct {
	sync.Mutex
	runtime     *Runtime
	enableCors  bool
	pullingPool map[string]struct{}
	pushingPool map[string]struct{}
	events      []utils.JSONMessage
	listeners   map[string]chan utils.JSONMessage
}
