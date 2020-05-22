package spacestore

import (
	"context"
	ipfslite "github.com/hsanjuan/ipfs-lite"
	"github.com/ipfs/go-cid"
	format "github.com/ipfs/go-ipld-format"
	"github.com/pkg/errors"
	"log"
	"strings"
)

// EntryNotFound error when a directory is not found
var EntryNotFound = errors.New("Directory entry not found")

// Memory Spacestore is an in-memory implementation fo the SpaceStore
// It uses the ipfs peer to fetch data and caches them in memory
// on restart all in-memory updates are lost
type MemorySpaceStore struct {
	peer      *ipfslite.Peer
	folderCid cid.Cid
	storage   map[string]*DirEntry
}

func NewMemoryStore(ctx context.Context) (*MemorySpaceStore, error) {
	ipfspeer, err := createIpfsPeer(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Error creating ipfs-lite peer")
	}

	folderCid, err := cid.Parse("QmZd7HwfHS15PdDeitZ7tdpfTq5CPFuQZysydQsTpUu8Kr")
	if err != nil {
		return nil, err
	}

	return &MemorySpaceStore{
		peer:      ipfspeer,
		folderCid: folderCid,
		storage:   make(map[string]*DirEntry),
	}, nil
}

func (m *MemorySpaceStore) Get(ctx context.Context, path string) (*DirEntry, error) {
	log.Printf("Get path: %s in folder %s", path, m.folderCid.String())
	folderOrFileName := m.folderCid.String()
	if entry, exists := m.storage[path]; exists && entry != nil {
		return entry, nil
	}

	// walk the path upward
	base := ""
	currentCid := m.folderCid
	trimmedPath := strings.TrimRight(path, "/")
	parts := strings.Split(trimmedPath, "/")
	var node format.Node
	for i, part := range parts {
		if i == 0 {
			base = "/"
			folderOrFileName = m.folderCid.String()
		} else {
			base += part
			folderOrFileName = part
		}

		// try and get from cache
		if entry, exists := m.storage[base]; exists && entry != nil {
			node = entry.node
			if i != 0 {
				base += "/"
			}
			continue
		}

		//log.Printf("Part '%s' of path %s", part, base)
		if base != "/" {
			// we are past base, so fetch cid from link
			if node == nil {
				log.Printf(
					"Should never happen, but nil node is used to check links for '%s' at path %s",
					part,
					base,
				)
				return nil, EntryNotFound
			}
			foundLink := false
			for _, link := range node.Links() {
				if link.Name == part {
					currentCid = link.Cid
					foundLink = true
					break
				}
			}

			if !foundLink {
				log.Printf("Entry '%s' not found in path %s", part, base)
				return nil, EntryNotFound
			}
		}
		//log.Printf("With cid %s", currentCid.String())

		// fetch from network
		newNode, err := m.peer.Get(ctx, currentCid)
		if err != nil {
			return nil, EntryNotFound
		}
		node = newNode

		// cache result in storage
		result := NewDirEntry(
			base,
			folderOrFileName,
			node,
			nil,
		)
		m.storage[base] = result

		if i != 0 {
			base += "/"
		}
	}

	if trimmedPath == "" {
		trimmedPath = "/"
	}
	return m.storage[trimmedPath], nil
}

func (m *MemorySpaceStore) GetChildren(ctx context.Context, path string) ([]*DirEntry, error) {
	log.Printf("Get children of path %s", path)
	dir, err := m.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	links := dir.node.Links()

	var result []*DirEntry
	for _, link := range links {
		name := link.Name
		node, err := m.peer.Get(ctx, link.Cid)
		if err != nil {
			log.Printf("Error fetching child %s of path %s: %s", name, path, err.Error())
			return nil, err
		}
		newEntry := NewDirEntry(
			path+"/"+name,
			name,
			node,
			nil, // will be fetch when stats are needed by DirEntry
		)
		result = append(result, newEntry)

		// cache it as well
		m.storage[path+"/"+name] = newEntry
	}

	return result, nil
}