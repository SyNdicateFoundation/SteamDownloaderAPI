from os import PathLike


class SteamCommand:
    _commands = []
    _force_install_dir: PathLike|str = None

    def __init__(self, force_install_dir: PathLike|str = None):
        self._commands = []
        self._force_install_dir = force_install_dir

    def set_force_install_dir(self, install_dir: PathLike|str):
        self._force_install_dir = install_dir
        return len(self._commands)

    def app_update(self, app_id: int, validate: bool = False, beta: str = '', beta_pass: str = ''):
        self._commands.append('+app_update {}{}{}{}'.format(
            app_id,
            ' validate' if validate else '',
            ' -beta {}'.format(beta) if beta else '',
            ' -betapassword {}'.format(beta_pass) if beta_pass else '',
        ))
        return len(self._commands) - 1

    def workshop_download_item(self, app_id: int, workshop_id: int, validate: bool = False):
        self._commands.append('+workshop_download_item {} {}{}'.format(
            app_id,
            workshop_id,
            ' validate' if validate else ''
        ))
        return len(self._commands) - 1

    def custom(self, cmd: str):
        self._commands.append(cmd)
        return len(self._commands) - 1

    def remove(self, idx):
        if 0 <= idx < len(self._commands) and self._commands[idx]:
            self._commands[idx] = None
            return True
        else:
            return False

    def get_cmd(self):
        return filter(None, self._commands)

    @property
    def install_dir(self):
        return self._force_install_dir

    @property
    def commands(self):
        return self._commands
