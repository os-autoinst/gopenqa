package main

import (
	"fmt"
	"os"

	"github.com/grisu48/gopenqa"
)

func main() {
	o3 := gopenqa.CreateO3Instance() // same as gopenqa.CreateInstance("https://openqa.opensuse.org")
	fmt.Println("Fetching job groups from openqa.opensuse.org ... ")
	if jgs, err := o3.GetJobGroups(); err != nil {
		fmt.Fprintf(os.Stderr, "error getting job groups: %s\n", err)
		os.Exit(1)
	} else {
		fmt.Printf("%d job groups fetched:\n", len(jgs))
		for _, jg := range jgs {
			jobs, err := o3.GetJobGroupJobs(jg.ID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching job groups for %s: %s\n", jg.Name, err)
			} else {
				fmt.Printf("  %4d %-40s %d jobs\n", jg.ID, jg.Name, len(jobs))
			}
		}
	}
}
