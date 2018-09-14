/*
 * Copyright (C) 2018 Waldemar Kozaczuk.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 *
 * This code implements the machanics of creating a ROFS file system
 * as described by comments in this Python code -
 * https://raw.githubusercontent.com/cloudius-systems/osv/master/scripts/gen-rofs-img.py.
 * The main public function used by upstream code is WriteRofsImage().
 */

package util

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	BLOCK_SIZE = 512

	DIR_MODE  = 0x4000
	REG_MODE  = 0x8000
	LINK_MODE = 0xA000
)

type RofsSuperBlock struct {
	Magic                    uint64
	Version                  uint64
	BlockSize                uint64
	StructureInfoFirstBlock  uint64
	StructureInfoBlocksCount uint64
	DirectoryEntriesCount    uint64
	SymlinksCount            uint64
	InodesCount              uint64
}

type RofsDirectoryEntry struct {
	InodeNumber uint64
	Filename    string
}

type RofsSymlink struct {
	Filename string
}

type RofsInode struct {
	Mode        uint64
	InodeNumber uint64
	DataOffset  uint64
	Count       uint64 // either file size in bytes or children count
}

type RofsFilesystem struct {
	SuperBlock             RofsSuperBlock
	DirectoryEntries       []*RofsDirectoryEntry
	Symlinks               []*RofsSymlink
	Inodes                 []*RofsInode
	DirectoryEntriesByPath map[string][]string
	CurrentBlock           int
}

func pad(buf *bytes.Buffer, count int) error {
	for ; count > 0; count-- {
		if err := buf.WriteByte(0); err != nil {
			return err
		}
	}
	return nil
}

func writeSuperBlock(imageFile *os.File, superBlock *RofsSuperBlock) error {
	buf := bytes.Buffer{}
	if err := binary.Write(&buf, binary.LittleEndian, superBlock); err != nil {
		return err
	}

	if err := pad(&buf, BLOCK_SIZE-buf.Len()); err != nil {
		return err
	}

	if _, err := imageFile.Seek(0, 0); err != nil {
		return err
	}

	_, err := imageFile.Write(buf.Bytes())
	return err
}

func ReadRofsSuperBlock(imageFile *os.File) (*RofsSuperBlock, error) {
	if _, err := imageFile.Seek(0, 0); err != nil {
		return nil, err
	}

	bytesArray := make([]byte, BLOCK_SIZE)
	if _, err := imageFile.Read(bytesArray); err != nil {
		return nil, err
	}

	superBlock := RofsSuperBlock{}
	buffer := bytes.NewBuffer(bytesArray)
	if err := binary.Read(buffer, binary.LittleEndian, &superBlock); err != nil {
		return nil, err
	}

	return &superBlock, nil
}

func writeString(buffer *bytes.Buffer, str string) error {
	if err := binary.Write(buffer, binary.LittleEndian, uint16(len(str))); err != nil {
		return err
	}
	for _, character := range str {
		if err := binary.Write(buffer, binary.LittleEndian, uint8(character)); err != nil {
			return err
		}
	}
	return nil
}

func writeDirectoryEntry(imageFile *os.File, entry *RofsDirectoryEntry) (int, error) {
	buf := bytes.Buffer{}
	if err := binary.Write(&buf, binary.LittleEndian, uint64(entry.InodeNumber)); err != nil {
		return 0, err
	}
	if err := writeString(&buf, entry.Filename); err != nil {
		return 0, err
	}

	return imageFile.Write(buf.Bytes())
}

func writeSymlink(imageFile *os.File, symlink *RofsSymlink) (int, error) {
	buf := bytes.Buffer{}
	if err := writeString(&buf, symlink.Filename); err != nil {
		return 0, err
	}

	return imageFile.Write(buf.Bytes())
}

func writeInode(imageFile *os.File, inode *RofsInode) (int, error) {
	buf := bytes.Buffer{}
	if err := binary.Write(&buf, binary.LittleEndian, inode); err != nil {
		return 0, err
	}

	return imageFile.Write(buf.Bytes())
}

func writeFile(filesystem *RofsFilesystem, imageFile *os.File, sourceFilePath string) (int, error) {
	sourceFile, err := os.Open(sourceFilePath)
	if err != nil {
		return 0, err
	}
	defer sourceFile.Close()

	buffer := make([]byte, BLOCK_SIZE)
	totalCount := 0
	for {
		count, readError := sourceFile.Read(buffer)
		if readError != nil && readError != io.EOF {
			return 0, readError
		}
		totalCount += count
		filesystem.CurrentBlock += 1
		//
		// Pad last block with 0 if EOF
		if readError == io.EOF {
			for ; count < BLOCK_SIZE; count++ {
				buffer[count] = 0
			}
		}

		if _, writeError := imageFile.Write(buffer); writeError != nil {
			return 0, writeError
		}
		if readError == io.EOF {
			break
		}
	}
	err = imageFile.Sync()
	return totalCount, err
}

func writeDirectory(filesystem *RofsFilesystem, imageFile *os.File, paths map[string]string,
	sourceDirectoryPath string, verbose bool) (int, int, error) {

	directoryEntries := filesystem.DirectoryEntriesByPath[sourceDirectoryPath]
	sort.Strings(directoryEntries)

	var thisDirectoryEntryInodes []*RofsDirectoryEntry
	for _, directoryEntry := range directoryEntries {
		//
		// Add new inode
		newInode := RofsInode{
			InodeNumber: uint64(len(filesystem.Inodes) + 1),
		}
		filesystem.Inodes = append(filesystem.Inodes, &newInode)
		//
		// Add new directory entry
		newDirectoryEntry := RofsDirectoryEntry{
			InodeNumber: newInode.InodeNumber,
			Filename:    directoryEntry,
		}
		thisDirectoryEntryInodes = append(thisDirectoryEntryInodes, &newDirectoryEntry)
		//
		// Check type of entry
		directoryEntryPath := filepath.Join(sourceDirectoryPath, directoryEntry)
		fi, err := os.Lstat(directoryEntryPath)
		if err != nil {
			return 0, 0, err
		}

		switch {
		case fi.Mode()&os.ModeSymlink == os.ModeSymlink:
			linkTarget, _ := os.Readlink(directoryEntryPath)
			if strings.HasPrefix(linkTarget, "/") || strings.HasPrefix(linkTarget, "..") {
				srcDir := filepath.Dir(directoryEntryPath)

				if linkTarget, err = filepath.Abs(filepath.Join(srcDir, linkTarget)); err != nil {
					return 0, 0, err
				}
				dst := paths[directoryEntryPath]
				linkTarget = strings.TrimPrefix(linkTarget, strings.TrimSuffix(directoryEntryPath, dst))
			}
			if verbose {
				fmt.Printf("Link %s to %s\n", paths[directoryEntryPath], linkTarget)
			}
			newInode.Mode = LINK_MODE
			newInode.DataOffset = uint64(len(filesystem.Symlinks))
			newInode.Count = uint64(1)
			//
			// Add new symlink entry
			newSymlinkEntry := RofsSymlink{
				Filename: linkTarget,
			}
			filesystem.Symlinks = append(filesystem.Symlinks, &newSymlinkEntry)

		case fi.Mode().IsDir():
			entriesCount, entriesIndex, err := writeDirectory(filesystem, imageFile, paths, directoryEntryPath, verbose)
			if err != nil {
				return 0, 0, err
			}
			newInode.Mode = DIR_MODE
			newInode.DataOffset = uint64(entriesIndex)
			newInode.Count = uint64(entriesCount)

		case fi.Mode().IsRegular():
			if verbose {
				fmt.Printf("Adding file: %s\n", paths[directoryEntryPath])
			}
			newInode.DataOffset = uint64(filesystem.CurrentBlock)
			bytesWritten, err := writeFile(filesystem, imageFile, directoryEntryPath)
			if err != nil {
				return 0, 0, err
			}
			newInode.Mode = REG_MODE
			newInode.Count = uint64(bytesWritten)
		}
	}
	thisDirectoryEntriesIndex := len(filesystem.DirectoryEntries)
	filesystem.DirectoryEntries = append(filesystem.DirectoryEntries, thisDirectoryEntryInodes...)
	return len(thisDirectoryEntryInodes), thisDirectoryEntriesIndex, nil
}

func writeFileSystem(filesystem *RofsFilesystem, imagePath string, paths map[string]string,
	sourceRootPath string, verbose bool) error {

	imageFile, err := os.Create(imagePath)
	if err != nil {
		return err
	}
	defer imageFile.Close()

	if verbose {
		fmt.Printf("Writing ROFS filesystem\n")
	}
	//
	// Write super block
	if err = writeSuperBlock(imageFile, &filesystem.SuperBlock); err != nil {
		return err
	}
	//
	// Create root inode
	rootInode := RofsInode{
		InodeNumber: uint64(len(filesystem.Inodes) + 1),
		Mode:        DIR_MODE,
	}
	filesystem.Inodes = append(filesystem.Inodes, &rootInode)

	entriesCount, entriesIndex, err := writeDirectory(filesystem, imageFile, paths, sourceRootPath, verbose)
	if err != nil {
		return err
	}

	systemStructureBlock := filesystem.CurrentBlock
	rootInode.DataOffset = uint64(entriesIndex)
	rootInode.Count = uint64(entriesCount)
	//
	// Write directory entries
	bytesWritten := 0
	for _, directoryEntry := range filesystem.DirectoryEntries {
		count, err := writeDirectoryEntry(imageFile, directoryEntry)
		if err != nil {
			return err
		}
		bytesWritten += count
	}
	//
	// Write symlinks
	for _, symlink := range filesystem.Symlinks {
		count, err := writeSymlink(imageFile, symlink)
		if err != nil {
			return err
		}
		bytesWritten += count
	}

	// Write inodes
	for _, inode := range filesystem.Inodes {
		count, err := writeInode(imageFile, inode)
		if err != nil {
			return err
		}
		bytesWritten += count
	}

	filesystem.SuperBlock.StructureInfoFirstBlock = uint64(systemStructureBlock)
	filesystem.SuperBlock.StructureInfoBlocksCount = uint64(bytesWritten / BLOCK_SIZE)
	if bytesWritten%BLOCK_SIZE > 0 {
		filesystem.SuperBlock.StructureInfoBlocksCount++
	}
	filesystem.SuperBlock.DirectoryEntriesCount = uint64(len(filesystem.DirectoryEntries))
	filesystem.SuperBlock.SymlinksCount = uint64(len(filesystem.Symlinks))
	filesystem.SuperBlock.InodesCount = uint64(len(filesystem.Inodes))

	if verbose {
		sb := filesystem.SuperBlock
		fmt.Printf("First block: %d, blocks count: %d\n", sb.StructureInfoFirstBlock, sb.StructureInfoBlocksCount)
		fmt.Printf("Directory entries count %d\n", sb.DirectoryEntriesCount)
		fmt.Printf("Symlinks count %d\n", sb.SymlinksCount)
		fmt.Printf("Inodes count %d\n", sb.InodesCount)
	}

	return writeSuperBlock(imageFile, &filesystem.SuperBlock)
}

func WriteRofsImage(imagePath string, paths map[string]string, sourceRootPath string, verbose bool) error {
	//
	// Create main fileystem structure to keep track of all information about
	// filesystem to be written to an image file
	filesystem := RofsFilesystem{
		SuperBlock: RofsSuperBlock{
			Magic:     0xDEADBEAD,
			Version:   1,
			BlockSize: BLOCK_SIZE,
		},
		DirectoryEntries:       []*RofsDirectoryEntry{},
		Symlinks:               []*RofsSymlink{},
		Inodes:                 []*RofsInode{},
		DirectoryEntriesByPath: make(map[string][]string),
		CurrentBlock:           1,
	}
	//
	// Create supporting map to allow to navigate directory entries
	for src, dest := range paths {
		if dest == "/" { // Skip root
			continue
		}
		//
		// Break src into directory path and basename and add to the DirectoryEntriesByPath
		directory, filename := filepath.Dir(src), filepath.Base(src)
		filesystem.DirectoryEntriesByPath[directory] = append(filesystem.DirectoryEntriesByPath[directory], filename)
	}
	return writeFileSystem(&filesystem, imagePath, paths, sourceRootPath, verbose)
}
