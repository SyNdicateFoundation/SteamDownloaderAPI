class SteamCMDException(Exception):
    def __init__(self, message=None, *args, **kwargs):
        self.message = message
        super(SteamCMDException, self).__init__(message, *args)

    def __unicode__(self):
        return repr(self.message)

    def __str__(self):
        return repr(self.message)


class SteamCMDDownloadException(SteamCMDException):
    def __init__(self, *args, **kwargs):
        super(SteamCMDDownloadException, self).__init__(*args, **kwargs)


class SteamCMDInstallException(SteamCMDException):
    def __init__(self, *args, **kwargs):
        super(SteamCMDInstallException, self).__init__(*args, **kwargs)
