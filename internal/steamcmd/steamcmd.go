package steamcmd

import (
	"fmt"
	"github.com/SyNdicateFoundation/SteamDownloaderAPI/internal/util"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

type SteamCMD struct {
	InstallPath string
	ExePath     string
	username    string
	password    string
}

func New(installPath string, username string, password string) (*SteamCMD, error) {
	exeName := "steamcmd"
	if runtime.GOOS == "windows" {
		exeName += ".exe"
	} else if runtime.GOOS == "linux" {
		exeName += ".sh"
	}

	absExePath, err := filepath.Abs(filepath.Join(installPath, exeName))
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for steamcmd: %w", err)
	}

	return &SteamCMD{
		InstallPath: installPath,
		ExePath:     absExePath,
		username:    username,
		password:    password,
	}, nil
}

func (s *SteamCMD) Install() error {
	if _, err := os.Stat(s.ExePath); err == nil {
		log.Println("✅ SteamCMD is already installed.")
		return nil
	}

	log.Println("Installing SteamCMD...")
	if runtime.GOOS == "windows" {
		return s.installWindows()
	}
	return s.installLinux()
}

func (s *SteamCMD) installWindows() error {
	url := "https://steamcdn-a.akamaihd.net/client/installer/steamcmd.zip"
	zipPath := filepath.Join(s.InstallPath, "steamcmd.zip")
	defer os.Remove(zipPath)

	if err := util.DownloadFile(url, zipPath); err != nil {
		return fmt.Errorf("failed to download steamcmd for Windows: %w", err)
	}

	if err := util.Unzip(zipPath, s.InstallPath); err != nil {
		return fmt.Errorf("failed to unzip steamcmd: %w", err)
	}

	return s.finalizeInstallation()
}

func (s *SteamCMD) installLinux() error {

	url := "https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz"
	tarPath := filepath.Join(s.InstallPath, "steamcmd.tar.gz")
	defer os.Remove(tarPath)

	if err := util.DownloadFile(url, tarPath); err != nil {
		return fmt.Errorf("failed to download steamcmd for Linux: %w", err)
	}

	if err := util.UntarGz(tarPath, s.InstallPath); err != nil {
		return fmt.Errorf("failed to untar steamcmd: %w", err)
	}

	if err := os.Chmod(s.ExePath, 0755); err != nil {
		return fmt.Errorf("failed to make steamcmd executable: %w", err)
	}

	return s.finalizeInstallation()
}

func (s *SteamCMD) finalizeInstallation() error {
	log.Println("💦 Finalizing SteamCMD installation (this may take a moment)...")
	cmd := exec.Command(s.ExePath, "+quit")
	cmd.Dir = s.InstallPath
	if err := cmd.Run(); err != nil {

		log.Printf("⚠️ SteamCMD quit with a non-zero exit code during finalization, this is often normal: %v", err)
	}
	log.Println("✅ SteamCMD installation complete.")
	return nil
}

func (s *SteamCMD) DownloadWorkshopItem(appID, workshopID int, validate bool) error {
	loginInfo := []string{"+login", "anonymous"}

	if s.username != "" {
		loginInfo = append(loginInfo, "+login", s.username, s.password)
	}

	args := append(
		loginInfo,
		[]string{"+workshop_download_item", fmt.Sprint(appID), fmt.Sprint(workshopID)}...,
	)

	if validate {
		args = append(args, "validate")
	}
	args = append(args, "+quit")

	cmd := exec.Command(s.ExePath, args...)
	cmd.Dir = s.InstallPath
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		time.Sleep(2 * time.Second)
		if errRetry := cmd.Run(); errRetry != nil {
			return fmt.Errorf("steamcmd execution failed after retry: %w", errRetry)
		}
	}

	return nil
}

func (s *SteamCMD) GetWorkshopContentPath(appID, workshopID int) string {
	return filepath.Join(s.InstallPath, "steamapps", "workshop", "content", fmt.Sprint(appID), fmt.Sprint(workshopID))
}
