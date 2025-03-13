package builder

func RelocateTree(root *Directory, block *uint32) {
	for entry := range root.Walk(false) {
		entry.Relocate(AllocateAndIncrementBlock(block, entry.PointerRecord().DataLength.RealValue()))
	}
}

func AllocateAndIncrementBlock(block *uint32, size uint32) uint32 {
	allocation := *block
	*block += (size + logicalBlockSize - 1) / logicalBlockSize

	return allocation
}
