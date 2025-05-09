package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

const DETACHED_PROCESS = 0x00000008

func main() {
	var mode int

	run_silent := flag.Bool("run", false, "run")
	flag.Parse()

	for {
		cls := exec.Command("cmd", "/c", "cls")
		cls.Stdout = os.Stdout
		cls.Run()

		if *run_silent {
			mode = 1
		} else {
			fmt.Print("Always run as admin please!\n" +
				"1) Run (or restart) Xray and set the hosts\n" +
				"2) Shutdown Xray and clear the hosts\n" +
				"3) Add to the startup\n" +
				"4) Remove from the startup\n" +
				"-> ")
			fmt.Scanln(&mode)
		}

		if mode == 1 { // run
			// check if xray dns pid exist then kill it
			if b, err := os.ReadFile(GetTruePath("pid1")); err == nil {
				// kill xray
				pid, _ := strconv.Atoi(string(b))
				KillXrayByPid(int32(pid))

				// remove pid file
				os.Remove(GetTruePath("pid1"))
			}

			// check if xray sni pid exist then kill it
			if b, err := os.ReadFile(GetTruePath("pid2")); err == nil {
				// kill xray
				pid, _ := strconv.Atoi(string(b))
				KillXrayByPid(int32(pid))

				// remove pid file
				os.Remove(GetTruePath("pid2"))
			}

			// run xray dns
			cmd := exec.Command(GetTruePath("xray.exe"), "run", "-c", GetTruePath("config-dns.jsonc"))
			cmd.SysProcAttr = &syscall.SysProcAttr{
				CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | DETACHED_PROCESS,
			}
			err := cmd.Start()
			if err != nil {
				fmt.Printf("\nError: %v\nPress enter...", err)
				fmt.Scanln()
				continue
			}

			// save xray dns pid
			err = os.WriteFile(GetTruePath("pid1"), []byte(strconv.Itoa(cmd.Process.Pid)), 0666)
			if err != nil {
				fmt.Printf("\nError: %v\nPress enter...", err)
				fmt.Scanln()
				continue
			}

			// run xray sni
			cmd = exec.Command(GetTruePath("xray.exe"), "run", "-c", GetTruePath("config-sni.jsonc"))
			cmd.SysProcAttr = &syscall.SysProcAttr{
				CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | DETACHED_PROCESS,
			}
			err = cmd.Start()
			if err != nil {
				fmt.Printf("\nError: %v\nPress enter...", err)
				fmt.Scanln()
				continue
			}

			// save xray sni pid
			err = os.WriteFile(GetTruePath("pid2"), []byte(strconv.Itoa(cmd.Process.Pid)), 0666)
			if err != nil {
				fmt.Printf("\nError: %v\nPress enter...", err)
				fmt.Scanln()
				continue
			}

			// flush dns
			cls := exec.Command("cmd", "/c", "ipconfig", "/flushdns")
			cls.Stdout = os.Stdout
			cls.Run()

			// // modify hosts
			// is_failed := false
			// for _, dom := range domains {
			// 	err = editHosts(0, dom)
			// 	if err != nil {
			// 		is_failed = true
			// 		break
			// 	}
			// }
			// if is_failed {
			// 	fmt.Printf("\nError: %v\n\n* Edit the hosts file manually or run as admin.\nPress enter...", err)
			// 	fmt.Scanln()
			// 	continue
			// }

			// success
			var intfc string
			for range 10 {
				if intfc, err = GetMainAdapterName(); err == nil {
					break
				}
				time.Sleep(500 * time.Millisecond)
			}
			time.Sleep(500 * time.Millisecond)
			SetDNS(intfc, "127.0.0.1", "")
			fmt.Println("\nSuccess!\nPress enter...")
			if *run_silent {
				return
			}
			fmt.Scanln()

		} else if mode == 2 { // shutdown
			// // modify hosts
			// var err error
			// is_failed := false
			// for _, dom := range domains {
			// 	err = editHosts(1, dom)
			// 	if err != nil {
			// 		is_failed = true
			// 		break
			// 	}
			// }
			// if is_failed {
			// 	if err != nil {
			// 		fmt.Printf("\nError: %v\n\n* Edit the hosts file manually or run as admin.\nPress enter...", err)
			// 	}
			// }

			// check and read xray dns pid file
			b, err := os.ReadFile(GetTruePath("pid1"))
			if err != nil {
				fmt.Println("\nXray is NOT running!\nSelect (1) to run it.")
			} else {
				// kill xray
				pid, _ := strconv.Atoi(string(b))
				err = KillXrayByPid(int32(pid))
				if err != nil {
					fmt.Printf("\nError: %v\nXray is NOT running!\nSelect (1) to run it.\n", err)
				}

				// remove pid file
				os.Remove(GetTruePath("pid1"))

				fmt.Println("\nSuccess!")
			}

			// check and read xray sni pid file
			b, err = os.ReadFile(GetTruePath("pid2"))
			if err != nil {
				fmt.Println("\nXray is NOT running!\nSelect (1) to run it.")
			} else {
				// kill xray
				pid, _ := strconv.Atoi(string(b))
				err = KillXrayByPid(int32(pid))
				if err != nil {
					fmt.Printf("\nError: %v\nXray is NOT running!\nSelect (1) to run it.\n", err)
				}

				// remove pid file
				os.Remove(GetTruePath("pid2"))

				fmt.Println("\nSuccess!")
			}

			// reset dns
			intfc, _ := GetMainAdapterName()
			SetDNS(intfc, "1.1.1.1", "1.0.0.1")

			// flush dns
			cls := exec.Command("cmd", "/c", "ipconfig", "/flushdns")
			cls.Stdout = os.Stdout
			cls.Run()

			fmt.Println("\nPress enter...")
			fmt.Scanln()

		} else if mode == 3 { // add to startup
			err := AddToStartup("--run")
			if err != nil {
				fmt.Println(err)
			}

			fmt.Println("\nPress enter...")
			fmt.Scanln()
		} else if mode == 4 { // remove from startup
			err := RemoveFromStartup()
			if err != nil {
				fmt.Println(err)
			}

			fmt.Println("\nPress enter...")
			fmt.Scanln()
		} else {
			fmt.Println("\nWrong select!")
			time.Sleep(1 * time.Second)
		}
	}

}

func KillXrayByPid(pid int32) error {
	proc, err := process.NewProcess(pid)
	if err != nil {
		return err
	}

	name, _ := proc.Name()
	if name != "xray.exe" {
		return process.ErrorProcessNotRunning
	}

	err = proc.Kill()
	if err != nil {
		return err
	}
	return nil
}

func EditHosts(mode int, domain string) error {
	newHosts := make([]string, 0)
	b, _ := os.ReadFile(`C:\Windows\System32\drivers\etc\hosts`)
	hosts := strings.Split(strings.TrimSpace(string(b)), "\n")

	if mode == 0 {
		for _, line := range hosts {
			// ignore comments
			if strings.HasPrefix(line, "#") {
				newHosts = append(newHosts, line)
				continue
			}

			// delete old xray-sni (127.0.0.1)
			if strings.TrimSpace(line) == "127.0.0.1 "+domain {
				continue
			}

			// find discord and comment it
			if strings.Contains(line, domain) {
				newHosts = append(newHosts, "#"+line)
				continue
			}

			newHosts = append(newHosts, line)
		}
		// add gateway.discord.gg record
		newHosts = append(newHosts, "127.0.0.1 "+domain)

		// write new hosts
		err := os.WriteFile(`C:\Windows\System32\drivers\etc\hosts`, []byte(strings.Join(newHosts, "\n")+"\n"), 0644)
		if err != nil {
			return err
		}

	} else if mode == 1 {
		for _, line := range hosts {
			// ignore comments
			if strings.HasPrefix(line, "#") {
				newHosts = append(newHosts, line)
				continue
			}

			// find discord and delete it
			if strings.Contains(line, "127.0.0.1 "+domain) {
				continue
			}

			newHosts = append(newHosts, line)
		}

		// write new hosts
		err := os.WriteFile(`C:\Windows\System32\drivers\etc\hosts`, []byte(strings.Join(newHosts, "\n")+"\n"), 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

// AddToStartup adds the current executable to the Windows startup.
func AddToStartup(args string) error {
	// Get the absolute path to the executable
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %v", err)
	}
	exeName := strings.TrimSuffix(filepath.Base(exePath), filepath.Ext(exePath))

	// Define the XML configuration for user logon (interactive only)
	taskXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Description>Run %s at user logon</Description>
    <Date>%s</Date>
  </RegistrationInfo>
  <Triggers>
    <LogonTrigger>
      <Enabled>true</Enabled>
    </LogonTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <LogonType>InteractiveToken</LogonType>
      <RunLevel>HighestAvailable</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <AllowHardTerminate>true</AllowHardTerminate>
    <StartWhenAvailable>true</StartWhenAvailable>
    <RunOnlyIfNetworkAvailable>false</RunOnlyIfNetworkAvailable>
    <Enabled>true</Enabled>
    <Hidden>false</Hidden>
    <RunOnlyIfIdle>false</RunOnlyIfIdle>
    <WakeToRun>false</WakeToRun>
    <ExecutionTimeLimit>P3D</ExecutionTimeLimit>
    <Priority>7</Priority>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>"%s"</Command>
      <Arguments>%s</Arguments>
      <WorkingDirectory>%s</WorkingDirectory>
    </Exec>
  </Actions>
</Task>`, exeName, time.Now().Format(time.RFC3339), exePath, args, filepath.Dir(exePath))

	// Write the XML to a temporary file
	xmlFile := GetTruePath("temp_task.xml")
	err = os.WriteFile(xmlFile, []byte(taskXML), 0644)
	if err != nil {
		return fmt.Errorf("failed to write task XML file: %v", err)
	}
	defer func() {
		if err := os.Remove(xmlFile); err != nil {
			fmt.Printf("Warning: failed to clean up temp file %s: %v\n", xmlFile, err)
		}
	}()

	// Create the scheduled task without explicit credentials
	cmd := exec.Command("schtasks", "/create", "/tn", exeName, "/xml", xmlFile, "/f")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to create scheduled task: %v", err)
	}

	fmt.Printf("Successfully added %s to startup via scheduled task at user logon.\n", exeName)
	return nil
}

// RemoveFromStartup removes the current executable from the Windows startup.
func RemoveFromStartup() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}
	exeName := strings.TrimSuffix(filepath.Base(exePath), filepath.Ext(exePath))

	cmd := exec.Command("schtasks", "/delete", "/tn", exeName, "/f")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to remove scheduled task: %v", err)
	}

	fmt.Printf("Successfully removed %s from startup.\n", exeName)
	return nil
}

// GetMainAdapterName returns the name of the main network adapter (interface)
// by checking for an active, non-loopback interface with a valid IP and gateway.
func GetMainAdapterName() (string, error) {
	// Get all network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get network interfaces: %v", err)
	}

	// Loop through all interfaces to find the "main" one
	for _, iface := range interfaces {
		// Skip interfaces that are down or loopback
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// Get the addresses for this interface
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		// Check if the interface has a valid IP address
		hasIP := false
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil || ipnet.IP.To16() != nil {
					hasIP = true
					break
				}
			}
		}

		if !hasIP {
			continue
		}

		// If this interface has a valid IP and is up, it's a good candidate
		// We can also check for a default gateway (Windows-specific logic may require external commands)
		return iface.Name, nil
	}

	return "", fmt.Errorf("no suitable network adapter found")
}

// SetDNS sets the primary and secondary DNS servers for the specified network interface.
func SetDNS(interfaceName, primaryDNS, secondaryDNS string) error {
	fmt.Printf("Setting dns for adapter %v", interfaceName)
	// Command to set the primary DNS server
	primaryCmd := exec.Command("netsh", "interface", "ipv4", "set", "dnsservers", interfaceName, "static", primaryDNS, "primary")
	primaryCmd.Stdout = os.Stdout
	primaryCmd.Stderr = os.Stderr

	// Execute the command to set the primary DNS
	err := primaryCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to set primary DNS: %v", err)
	}

	// If a secondary DNS is provided, set it as well
	if secondaryDNS != "" {
		secondaryCmd := exec.Command("netsh", "interface", "ipv4", "add", "dnsservers", interfaceName, secondaryDNS, "index=2")
		secondaryCmd.Stdout = os.Stdout
		secondaryCmd.Stderr = os.Stderr

		// Execute the command to set the secondary DNS
		err := secondaryCmd.Run()
		if err != nil {
			return fmt.Errorf("failed to set secondary DNS: %v", err)
		}
	}

	fmt.Printf("DNS servers set successfully for interface %s\n", interfaceName)
	return nil
}

// ResetDNS resets the DNS settings to obtain DNS automatically (DHCP).
func ResetDNS(interfaceName string) error {
	resetCmd := exec.Command("netsh", "interface", "ipv4", "set", "dnsservers", interfaceName, "source=dhcp")
	resetCmd.Stdout = os.Stdout
	resetCmd.Stderr = os.Stderr

	err := resetCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to reset DNS: %v", err)
	}

	fmt.Printf("DNS settings reset to DHCP for interface %s\n", interfaceName)
	return nil
}

func GetTruePath(filename string) string {
	// Get the path to the executable
	exePath, err := os.Executable()
	if err != nil {
		fmt.Println("Error getting executable path:", err)
		return filename
	}

	// Get the directory containing the executable
	exeDir := filepath.Dir(exePath)

	os.Chdir(exeDir)

	// Construct a path relative to the executable
	return filepath.Join(exeDir, filename)
}
