/*
 * This is a example CLI tool for demonstrating the usage of gopenqa. This has no further purposes and is not intended for productive usage
 */
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/grisu48/gopenqa"
)

type Config struct {
	Remote    string
	ApiKey    string
	ApiSecret string
	Verbose   bool
	NoPrompt  bool
}

var cf Config
var instance gopenqa.Instance

func (cf *Config) ApplyDefaults() {
	cf.Remote = "https://openqa.opensuse.org"
	cf.ApiKey = ""
	cf.ApiSecret = ""
	cf.Verbose = false
	cf.NoPrompt = false
}

func extractIntegers(args []string) ([]int, []string) {
	ids := make([]int, 0)
	rem := make([]string, 0)

	for _, arg := range args {
		if id, err := strconv.Atoi(arg); err == nil {
			ids = append(ids, id)
		} else {
			rem = append(rem, arg)
		}
	}
	return ids, rem
}

func prompt(msg string) string {
	if msg != "" {
		fmt.Print(msg)
		os.Stdout.Sync()
	}
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(line)
}

// Replace some magic by their remote
func magicRemote(remote string) string {
	// Skip if remote
	if strings.HasPrefix(remote, "http://") || strings.HasPrefix(remote, "https://") {
		return remote
	}
	if remote == "" || remote == "ooo" || remote == "o3" {
		return "https://openqa.opensuse.org"
	} else if remote == "osd" {
		return "http://openqa.suse.de"
	} else if remote == "duck" {
		return "http://duck-norris.qam.suse.de"
	}
	return remote
}

func usage() {
	var cf Config // Create new config with defaults to display them
	cf.ApplyDefaults()
	fmt.Printf("Usage: %s [OPTIONS] ENTITY [METHOD] [COMMAND]\n", os.Args[0])
	fmt.Println("OPTIONS")
	fmt.Printf("  -r, --remote INSTANCE                           Define the openQA instance (default: %s)\n", cf.Remote)
	fmt.Println("  -k, --apikey KEY                               Set APIKEY for instance")
	fmt.Println("  -s, --apisecret SECRET                         Set APISECRET for instance")
	fmt.Println("  -v, --verbose                                  Verbose run")
	fmt.Println("  -y                                             No prompt")
	fmt.Println("")
	fmt.Println("ENTITY")
	fmt.Println("")
	fmt.Println("  job [ID]")
	fmt.Println("  jobs [IDS...]")
	fmt.Println("  jobgroup(s)")
	fmt.Println("  machine(s)")
	fmt.Println("  product(s) | medium(s)")
	fmt.Println("  parentgroup(s)")
	fmt.Println("  comments")
	fmt.Println("  jobstate")
}

func parseArgs(args []string) (string, []string, error) {
	entity := ""
	commands := make([]string, 0)

	n := len(args)
	for i := 0; i < n; i++ {
		arg := args[i]
		if arg == "" {
			continue
		}
		if arg[0] == '-' {
			if arg == "-h" || arg == "--help" {
				usage()
				os.Exit(0)
			} else if arg == "-r" || arg == "--remote" || arg == "--openqa" {
				i++
				cf.Remote = magicRemote(args[i])

			} else if arg == "-k" || arg == "--apikey" {
				i++
				cf.ApiKey = args[i]
			} else if arg == "-s" || arg == "--apisecret" {
				i++
				cf.ApiSecret = args[i]
			} else if arg == "-v" || arg == "--verbose" {
				cf.Verbose = true
			} else if arg == "-y" || arg == "--yes" {
				cf.NoPrompt = true
			} else {
				return entity, args, fmt.Errorf("Invalid argument: %s", arg)
			}
		} else {
			commands = append(commands, arg)
		}
	}

	if len(commands) == 0 {
		return "", commands, fmt.Errorf("no entity given")
	} else {
		entity := commands[0]
		commands = commands[1:]
		return entity, commands, nil
	}
}

/* Read machines from stdin */
func readMachines(filename string) ([]gopenqa.Machine, error) {
	var data []byte
	var err error

	if filename == "" {
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			machines := make([]gopenqa.Machine, 0)
			return machines, err
		}
	} else {
		// TODO: Don't use io.ReadAll
		if file, err := os.Open(filename); err != nil {
			return make([]gopenqa.Machine, 0), err
		} else {
			defer file.Close()
			data, err = io.ReadAll(file)
			if err != nil {
				return make([]gopenqa.Machine, 0), err
			}
		}
	}

	// First try to read a single machine
	var machine gopenqa.Machine
	if err := json.Unmarshal(data, &machine); err == nil {
		machines := make([]gopenqa.Machine, 0)
		machines = append(machines, machine)
		return machines, nil
	}

	// Then try to read a machine array
	var machines []gopenqa.Machine
	if err := json.Unmarshal(data, &machines); err == nil {
		return machines, err
	}

	machines = make([]gopenqa.Machine, 0)
	return machines, fmt.Errorf("invalid input format")
}

/* Read products from stdin */
func readProducts(filename string) ([]gopenqa.Product, error) {
	var data []byte
	var err error

	if filename == "" {
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			return make([]gopenqa.Product, 0), err
		}
	} else {
		// TODO: Don't use io.ReadAll
		if file, err := os.Open(filename); err != nil {
			return make([]gopenqa.Product, 0), err
		} else {
			defer file.Close()
			data, err = io.ReadAll(file)
			if err != nil {
				return make([]gopenqa.Product, 0), err
			}
		}
	}

	// First try to read a single machine
	var product gopenqa.Product
	if err := json.Unmarshal(data, &product); err == nil {
		products := make([]gopenqa.Product, 0)
		products = append(products, product)
		return products, nil
	}

	// Then try to read a machine array
	var products []gopenqa.Product
	if err := json.Unmarshal(data, &products); err == nil {
		return products, err
	}

	products = make([]gopenqa.Product, 0)
	return products, fmt.Errorf("invalid input format")
}

/* Read job groups from stdin */
func readJobGroups(filename string) ([]gopenqa.JobGroup, error) {
	var data []byte
	var err error

	if filename == "" {
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			return make([]gopenqa.JobGroup, 0), err
		}
	} else {
		// TODO: Don't use io.ReadAll
		if file, err := os.Open(filename); err != nil {
			return make([]gopenqa.JobGroup, 0), err
		} else {
			defer file.Close()
			data, err = io.ReadAll(file)
			if err != nil {
				return make([]gopenqa.JobGroup, 0), err
			}
		}
	}

	// First try to read a single machine
	var jobgroup gopenqa.JobGroup
	if err := json.Unmarshal(data, &jobgroup); err == nil {
		jobgroups := make([]gopenqa.JobGroup, 0)
		jobgroups = append(jobgroups, jobgroup)
		return jobgroups, nil
	}

	// Then try to read a machine array
	var jobgroups []gopenqa.JobGroup
	if err := json.Unmarshal(data, &jobgroups); err == nil {
		return jobgroups, err
	}

	jobgroups = make([]gopenqa.JobGroup, 0)
	return jobgroups, fmt.Errorf("invalid input format")
}

func printJson(data interface{}) error {
	// Print as json
	if buf, err := json.Marshal(data); err != nil {
		return err
	} else {
		fmt.Println(string(buf))
		return nil
	}
}

func runJobTemplates(args []string) error {
	method := "GET"

	if len(args) > 0 {
		// get method
	}

	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "GET" {
		fmt.Fprintf(os.Stderr, "Getting job templates ... ")
		templates, err := instance.GetJobTemplates()
		if err != nil {
			return err
		}
		return printJson(templates)
	} else {
		return fmt.Errorf("invalid method: %s", method)
	}
}

func postMachines(args []string) error {
	files := args
	if len(files) == 0 {
		files = append(files, "")
	}

	for _, filename := range files {
		if machines, err := readMachines(filename); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		} else {
			for _, machine := range machines {
				if machine, err := instance.PostMachine(machine); err != nil {
					return err
				} else {
					fmt.Printf("Posted machine %d %s:%s\n", machine.ID, machine.Name, machine.Backend)
				}
			}
		}
	}

	return nil
}

func postJobGroups(args []string) error {
	files := args
	if len(files) == 0 {
		files = append(files, "")
	}

	for _, filename := range files {
		if jobgroups, err := readJobGroups(filename); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		} else {
			for _, jobgroup := range jobgroups {
				if jobgroup, err := instance.PostJobGroup(jobgroup); err != nil {
					return err
				} else {
					fmt.Printf("Posted job group %d %s\n", jobgroup.ID, jobgroup.Name)
				}
			}
		}
	}

	return nil
}

func postParentJobGroups(args []string) error {
	files := args
	if len(files) == 0 {
		files = append(files, "")
	}

	for _, filename := range files {
		if jobgroups, err := readJobGroups(filename); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		} else {
			for _, jobgroup := range jobgroups {
				if jobgroup, err := instance.PostParentJobGroup(jobgroup); err != nil {
					return err
				} else {
					fmt.Printf("Posted parent job group %d %s\n", jobgroup.ID, jobgroup.Name)
				}
			}
		}
	}

	return nil
}

func postProduct(args []string) error {
	files := args
	if len(files) == 0 {
		files = append(files, "")
	}

	for _, filename := range files {
		if products, err := readProducts(filename); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		} else {
			for _, product := range products {
				if product, err := instance.PostProduct(product); err != nil {
					return err
				} else {
					fmt.Printf("Posted product %d\n", product.ID)
				}
			}
		}
	}
	return nil
}

func runMachines(args []string) error {
	method := "GET"

	if len(args) > 0 {
		// get method
		method = args[0]
		args = args[1:]
	}

	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "GET" {
		if machines, err := instance.GetMachines(); err != nil {
			return err
		} else {
			return printJson(machines)
		}
	} else if method == "POST" {
		return postMachines(args)
	} else if method == "DELETE" {
		ids, _ := extractIntegers(args)
		if len(ids) == 0 {
			fmt.Fprintf(os.Stderr, "Missing machine ids\n")
		} else {
			for _, id := range ids {
				if err := instance.DeleteMachine(id); err != nil {
					return err
				} else {
					fmt.Printf("Deleted machine %d\n", id)
				}
			}
		}
		return nil
	} else if method == "CLEAR" {
		if !cf.NoPrompt {
			fmt.Println("DANGER ZONE !!")
			fmt.Println("Are you sure you want to delete ALL machines? THERE WILL BE NO UNDO, if you are hesitant then stop NOW.")
			if prompt("Type uppercase 'yes' to continue: ") != "YES" {
				return fmt.Errorf("cancelled")
			}
		}

		// Get machines and then delete them one by one
		if cf.Verbose {
			fmt.Println("Fetching machines ... ")
		}
		if machines, err := instance.GetMachines(); err != nil {
			return err
		} else {
			for i, machine := range machines {
				id := machine.ID
				if err := instance.DeleteMachine(id); err != nil {
					return err
				} else {
					fmt.Printf("[%d/%d] Deleted machine %d %s:%s\n", i, len(machines), machine.ID, machine.Name, machine.Backend)
				}
			}
		}

		return nil
	} else {
		return fmt.Errorf("invalid method: %s", method)
	}
}

func runMachine(args []string) error {
	method := "GET"
	ids, args := extractIntegers(args)

	if len(args) > 0 {
		// get method
		method = args[0]
		args = args[1:]
	}

	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "GET" {
		for _, id := range ids {
			if machines, err := instance.GetMachine(id); err != nil {
				return err
			} else {
				if err := printJson(machines); err != nil {
					return err
				}
			}
		}
		return nil
	} else if method == "POST" {
		return postMachines(args)
	} else {
		return fmt.Errorf("invalid method: %s", method)
	}
}

func runJobGroups(args []string) error {
	method := "GET"

	if len(args) > 0 {
		// get method
		method = args[0]
		args = args[1:]
	}

	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "GET" {
		if machines, err := instance.GetJobGroups(); err != nil {
			return err
		} else {
			return printJson(machines)
		}
	} else if method == "POST" {
		return postJobGroups(args)
	} else if method == "CLEAR" {
		if !cf.NoPrompt {
			fmt.Println("DANGER ZONE !!")
			fmt.Println("Are you sure you want to delete ALL job groups? THERE WILL BE NO UNDO, if you are hesitant then stop NOW.")
			fmt.Println("Deleting job groups also means to delete all attached jobs!")
			if prompt("Type uppercase 'yes' to continue: ") != "YES" {
				return fmt.Errorf("cancelled")
			}
		}

		if jobgroups, err := instance.GetJobGroups(); err != nil {
			return err
		} else {
			fmt.Printf("Delete %d job groups and their jobs ... \n", len(jobgroups))
			for i, jobgroup := range jobgroups {
				id := jobgroup.ID
				if err := instance.DeleteJobTemplate(id); err != nil {
					return err
				}
				if err := instance.DeleteJobGroupJobs(id); err != nil {
					return err
				}
				if err := instance.DeleteJobGroup(id); err != nil {
					return err
				}

				fmt.Printf("[%d/%d] Deleted job group %d %s\n", i, len(jobgroups), jobgroup.ID, jobgroup.Name)

			}
		}

		return nil
	} else {
		return fmt.Errorf("invalid method: %s", method)
	}
}

func runParentGroups(args []string) error {
	method := "GET"

	if len(args) > 0 {
		// get method
		method = args[0]
		args = args[1:]
	}

	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "GET" {
		if machines, err := instance.GetParentJobGroups(); err != nil {
			return err
		} else {
			return printJson(machines)
		}
	} else if method == "POST" {
		return postParentJobGroups(args)
	} else {
		return fmt.Errorf("invalid method: %s", method)
	}
}

func runJobGroup(args []string) error {
	method := "GET"

	if len(args) > 0 {
		// get method
		method = args[0]
		args = args[1:]
	}

	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "GET" {
		ids, args := extractIntegers(args)
		if len(args) > 0 {
			return fmt.Errorf("too many arguments")
		}
		for _, id := range ids {
			if jobgroup, err := instance.GetJobGroup(id); err != nil {
				return err
			} else {
				if err := printJson(jobgroup); err != nil {
					return err
				}
			}
		}
		return nil
	} else if method == "POST" {
		return postJobGroups(args)
	} else if method == "DELETE" {
		ids, args := extractIntegers(args)
		if len(args) > 0 {
			return fmt.Errorf("invalid arguments")
		}
		for _, id := range ids {
			if err := instance.DeleteJobGroup(id); err != nil {
				return err
			}
			fmt.Printf("Deleted job group %d\n", id)
		}
		return nil
	} else {
		return fmt.Errorf("invalid method: %s", method)
	}
}

func runParentGroup(args []string) error {
	method := "GET"

	if len(args) > 0 {
		// get method
		method = args[0]
		args = args[1:]
	}

	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "GET" {
		ids, args := extractIntegers(args)
		if len(args) > 0 {
			return fmt.Errorf("too many arguments")
		}
		for _, id := range ids {
			if jobgroup, err := instance.GetParentJobGroup(id); err != nil {
				return err
			} else {
				if err := printJson(jobgroup); err != nil {
					return err
				}
			}
		}
		return nil
	} else if method == "POST" {
		return postParentJobGroups(args)
	} else {
		return fmt.Errorf("invalid method: %s", method)
	}
}

func runProducts(args []string) error {
	method := "GET"

	if len(args) > 0 {
		// get method
		method = args[0]
		args = args[1:]
	}

	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "GET" {
		products, err := instance.GetProducts()
		if err != nil {
			return err
		}
		if err := printJson(products); err != nil {
			return err
		}
		return nil
	} else if method == "POST" {
		return postProduct(args)
	} else {
		return fmt.Errorf("invalid method: %s", method)
	}
}

func runProduct(args []string) error {
	method := "GET"

	if len(args) > 0 {
		// get method
		method = args[0]
		args = args[1:]
	}

	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "GET" {
		ids, _ := extractIntegers(args)
		if len(ids) == 0 {
			return fmt.Errorf("missing product ids")
		}
		for _, id := range ids {
			product, err := instance.GetProduct(id)
			if err != nil {
				return err
			}

			if err := printJson(product); err != nil {
				return err
			}
		}
		return nil
	} else if method == "POST" {
		return postProduct(args)
	} else {
		return fmt.Errorf("invalid method: %s", method)
	}
}

func runComments(args []string) error {
	method := "GET"

	if len(args) < 1 {
		return fmt.Errorf("not enough arguments")
	}
	// get method
	var id int64
	if len(args) == 1 {
		method = "GET"
		id, _ = strconv.ParseInt(args[0], 10, 64)
	} else {
		method = strings.ToUpper(strings.TrimSpace(args[0]))
		id, _ = strconv.ParseInt(args[1], 10, 64)
		args = args[2:]
	}
	if id <= 0 {
		return fmt.Errorf("invalid ID")
	}

	if method == "GET" {
		comments, err := instance.GetComments(id)
		if err != nil {
			return err
		}
		if err := printJson(comments); err != nil {
			return err
		}
		return nil
	} else {
		return fmt.Errorf("Method %s is not (yet) supported", method)
	}
}

func runJobState(args []string) error {
	var id int64
	if len(args) < 1 {
		return fmt.Errorf("missing argument: job")
	} else if len(args) > 1 {
		return fmt.Errorf("too many arguments")
	}

	id, _ = strconv.ParseInt(args[0], 10, 64)
	if id <= 0 {
		return fmt.Errorf("invalid ID")
	}

	state, err := instance.GetJobState(id)
	if err != nil {
		return err
	}
	if state.BlockedBy > 0 {
		fmt.Printf("Blocked by %d\n", state.BlockedBy)
	}
	if state.State == "done" {
		fmt.Println(state.Result)
	} else {
		fmt.Println(state.State)
	}
	return nil
}

func runJob(args []string) error {

	var id int64
	if len(args) < 1 {
		return fmt.Errorf("missing argument: job")
	} else if len(args) > 1 {
		return fmt.Errorf("too many arguments")
	}

	id, _ = strconv.ParseInt(args[0], 10, 64)
	if id <= 0 {
		return fmt.Errorf("invalid ID")
	}

	job, err := instance.GetJobFollow(id)
	if err != nil {
		return err
	}
	fmt.Println(job.String())
	return nil
}

func runJobs(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing argument: jobs")
	}
	ids := make([]int64, 0)

	for _, arg := range args {
		id, err := strconv.ParseInt(arg, 10, 64)
		if id <= 0 || err != nil {
			return fmt.Errorf("invalid ID")
		}
		ids = append(ids, id)
	}

	jobs, err := instance.GetJobsFollow(ids)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		fmt.Printf("%s\n", job.String())
	}
	return nil
}

func main() {
	cf.ApplyDefaults()

	if len(os.Args) <= 1 {
		usage()
		os.Exit(1)
	}

	entity, command, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	instance = gopenqa.CreateInstance(cf.Remote)
	instance.SetApiKey(cf.ApiKey, cf.ApiSecret)
	instance.SetVerbose(cf.Verbose)
	if entity == "h" || entity == "help" {
		usage()
		os.Exit(0)
	} else if entity == "jobtemplates" || entity == "templates" {
		if err := runJobTemplates(command); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
	} else if entity == "machines" {
		if err := runMachines(command); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
	} else if entity == "machine" {
		if err := runMachine(command); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
	} else if entity == "products" || entity == "mediums" {
		if err := runProducts(command); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
	} else if entity == "product" || entity == "medium" {
		if err := runProduct(command); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
	} else if entity == "jobgroups" || entity == "job_groups" {
		if err := runJobGroups(command); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
	} else if entity == "jobgroup" || entity == "job_group" {
		if err := runJobGroup(command); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
	} else if entity == "parentgroup" || entity == "parent_group" || entity == "parent_job_group" || entity == "parentjobgroup" {
		if err := runParentGroup(command); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
	} else if entity == "parentgroups" || entity == "parent_groups" || entity == "parent_job_groups" || entity == "parentjobgroups" {
		if err := runParentGroups(command); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
	} else if entity == "comments" {
		if err := runComments(command); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
	} else if entity == "jobstate" || entity == "state" {
		if err := runJobState(command); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
	} else if entity == "job" {
		if err := runJob(command); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
	} else if entity == "jobs" {
		if err := runJobs(command); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Invalid entity: %s\n", entity)
		os.Exit(1)
	}

}
