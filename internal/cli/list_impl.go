package cli

func init() {
	listCmd.RunE = runPipList
}
