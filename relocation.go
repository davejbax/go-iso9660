package iso9660

func relocateTree(root *directory, block *uint32) {
	for entry := range root.Walk() {
		entry.Relocate(allocateAndIncrementBlock(block, entry.DataLength()))
	}
}

func allocateAndIncrementBlock(block *uint32, size uint32) uint32 {
	allocation := *block
	*block += (size + logicalBlockSize - 1) / logicalBlockSize

	return allocation
}
