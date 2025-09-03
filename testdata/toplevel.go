  var commitChanges bool
  
  flag.BoolVar(&commitChanges, "commit-changes", false, "instructs the tool to commit changes to the data store")
  
  if commitChanges {
    if _, err := tx.Commit(); err != nil {
      log.Fatalf("could no commit changes: %v", err)
    }
  }
