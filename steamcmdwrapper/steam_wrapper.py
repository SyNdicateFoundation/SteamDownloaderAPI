from asyncio import create_subprocess_shell, to_thread
from getpass import getpass
from pathlib import Path
from platform import system
from shlex import join as shlex_join

from aiofiles import stdout
from aiofiles.os import remove, makedirs
from aiofiles.tempfile import NamedTemporaryFile
from httpx import AsyncClient

from steamcmdwrapper.exceptions import SteamCMDException, SteamCMDDownloadException, SteamCMDInstallException
from steamcmdwrapper.steam_command import SteamCommand

package_links = {
    "Windows": {
        "url": "https://steamcdn-a.akamaihd.net/client/installer/steamcmd.zip",
        "extension": ".exe",
        "d_extension": ".zip"
    },
    "Linux": {
        "url": "https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz",
        "extension": ".sh",
        "d_extension": ".tar.gz"
    }
}


class SteamCMDWrapper:
    _installation_path = ""
    _uname = "anonymous"
    _passw = ""

    def __init__(self, installation_path):
        self._installation_path = Path(installation_path)

        if not self._installation_path.is_dir():
            raise SteamCMDInstallException("""
            No valid directory found at {}.
            Please make sure that the directory is correct.
            """.format(self._installation_path))

        self._prepare_installation()

    def _prepare_installation(self):
        self.platform = system()
        if self.platform not in ["Windows", "Linux"]:
            raise SteamCMDException(f"Non supported operating system. Expected Windows or Linux, got {self.platform}")

        self.steamcmd_url = package_links[self.platform]["url"]
        self._exe = self._installation_path / ("steamcmd" + package_links[self.platform]["extension"])

    async def _download(self, file: NamedTemporaryFile):
        async with AsyncClient() as client:
            resp = await client.get(self.steamcmd_url)
            resp.raise_for_status()

            async for chunk in resp.aiter_bytes():
                await file.write(chunk)

    async def _extract_steamcmd(self, file: str):
        if self.platform == 'Windows':
            from zipfile import ZipFile

            if not self._installation_path.exists():
                await makedirs(self._installation_path)

            with ZipFile(file, 'r') as zip_ref:
                await to_thread(zip_ref.extractall, self._installation_path,
                                members=zip_ref.infolist(),
                                pwd=None)

        elif self.platform == 'Linux':
            from tarfile import open as open_tarfile

            if not self._installation_path.exists():
                await makedirs(self._installation_path)

            with open_tarfile(file) as tar:
                # noinspection PyTypeChecker
                await to_thread(tar.extractall, self._installation_path)

        else:
            raise SteamCMDException(
                'The operating system is not supported.'
                f'Expected Linux or Windows, received: {self.platform}'
            )

        await remove(file)

    @staticmethod
    async def _print_log(*message):
        for msg in message:
            await stdout.write(f"{msg} ")
        await stdout.write("\r\n")
        await stdout.flush()

    async def install(self, force: bool = False):
        if self._exe.is_file() and not force:
            raise SteamCMDException(
                'Steamcmd is already installed. Reinstall is not necessary.'
                'Use force=True to override.'
            )

        await self._print_log("Installing SteamCMD")

        async with NamedTemporaryFile("wb+", delete=False) as f:
            await self._download(f)

        await self._extract_steamcmd(f.name)

        if not (proc := await create_subprocess_shell(shlex_join([self._exe.as_posix(), "+quit"]),
                                                      cwd=self._installation_path)):
            raise SteamCMDException("Failed to start steamcmd")

        match code := await proc.wait():
            case 0:
                await self._print_log("SteamCMD successfully installed")
                return
            case 7:
                await self._print_log(
                    "SteamCMD has returned error code 7 on fresh installation",
                    "",
                    "Not sure why this crashed,",
                    "long live steamcmd and it's non existent documentation..",
                    "It should be fine nevertheless")
            case _:
                raise SteamCMDInstallException(f"Failed to install, check error code {code}")

        await remove(self._exe)

    async def login(self, uname: str = None, passw: str = None):
        self._uname = uname if uname else input("Please enter steam username: ")
        self._passw = passw if passw else getpass("Please enter steam password: ")

        sc = SteamCommand()

        return await self.execute(sc)

    def app_update(self, app_id: int, install_dir: str = None, validate: bool = None, beta: str = None,
                   betapassword: str = None):

        sc = SteamCommand(install_dir)
        sc.app_update(app_id, validate, beta, betapassword)

        self._print_log(
            f"Downloading item {app_id}",
            f"into {install_dir} with validate set to {validate}")
        return self.execute(sc)

    async def workshop_update(self, app_id: int, workshop_id: int, install_dir: Path = None, validate: bool = None,
                              n_tries: int = 5):
        sc = SteamCommand(install_dir)
        sc.workshop_download_item(app_id, workshop_id, validate)
        await self.execute(sc, n_tries)

    async def execute(self, cmd: SteamCommand, n_tries: int = 1):
        if n_tries == 0:
            raise SteamCMDDownloadException(
                """Error executing command, max number of timeout tries exceeded!
                Consider increasing the n_tries parameter if the download is
                particularly large"""
            )

        params = [
            self._exe.as_posix(),
            f"+force_install_dir {cmd.install_dir}" if cmd.install_dir else "",
            f"+login {self._uname} {self._passw}",
            *cmd.get_cmd(),
            "+quit",
        ]

        await self._print_log("Parameters used:", shlex_join(params))

        if not (proc := await create_subprocess_shell(shlex_join(params), cwd=self._installation_path)):
            raise SteamCMDException("Failed to start steamcmd")

        match code := await proc.wait():
            case 0:
                await self._print_log("SteamCMD successfully executed")
                return
            case 10:
                await self._print_log(f"Download timeout! Tries remaining: {n_tries}. Retrying...")
                return self.execute(cmd, n_tries - 1)
            case 134:
                await self._print_log(f"SteamCMD errored! Tries remaining: {n_tries}. Retrying...")
                return self.execute(cmd, n_tries - 1)
            case _:
                raise SteamCMDException(f"Steamcmd was unable to run. exit code was {code}")
