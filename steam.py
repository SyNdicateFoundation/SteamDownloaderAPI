from asyncio import run
from contextlib import suppress
from os import getcwd
from pathlib import Path
from re import compile
from typing import List

from aiofiles import open as async_open
from cache import AsyncTTL
from httpx import AsyncClient
from zipstream import AioZipStream

from steamcmdwrapper import SteamCMDWrapper, SteamCMDException, SteamCommand

steamcmd_path = Path(getcwd()) / "steamcmd"

steam_path = steamcmd_path / "steam"
steam_path.mkdir(exist_ok=True, parents=True)

steam = SteamCMDWrapper(steamcmd_path)

with suppress(SteamCMDException):
    run(steam.install())

workshop_pattern = compile(rb'<a\s+href="https://steamcommunity\.com/sharedfiles/filedetails/\?id=('
                           rb'\d+)">\s*<div\s+class="workshopItemTitle">([^<]+)</div>\s*</a>')
collection_title_pattern = compile(rb'<div\s+class=\"workshopItemTitle\">([^<]+)</div>')


@AsyncTTL(time_to_live=60, maxsize=40 << 10)
async def get_collection_items(workshop_id) -> tuple[str, list[tuple[int, str]]]:
    var = []
    title = ""

    async with AsyncClient() as client:
        response = await client.get(f"https://steamcommunity.com/sharedfiles/filedetails/?id={workshop_id}")

        response.raise_for_status()

        content = await response.aread()

        title = collection_title_pattern.search(content)
        cols = workshop_pattern.findall(content)

        title = title.group(1).decode()

        if not cols or not title:
            return title, var

        for coll in cols:
            var.append((int(coll[0]), coll[1].decode()))

    return title, var


@AsyncTTL(time_to_live=60, maxsize=40 << 10)
async def get_workshop_name(workshop_id):
    title = ""

    async with AsyncClient() as client:
        response = await client.get(f"https://steamcommunity.com/sharedfiles/filedetails/?id={workshop_id}")

        response.raise_for_status()

        content = await response.aread()

        title = collection_title_pattern.search(content)
        if not title:
            return None

        title = title.group(1).decode()

    return title


async def workshop_download(app_id, workshop_id, validate=True):
    workshop_name = await get_workshop_name(workshop_id)
    if not workshop_name:
        return None

    workshop_download_path = (steam_path / "steamapps" / "workshop" / "content" / str(app_id) / str(workshop_id))
    workshop_zip = workshop_download_path.as_posix() + "%s .zip" % workshop_name

    if not workshop_download_path.exists():
        try:
            await steam.workshop_update(app_id, workshop_id, steam_path, validate, n_tries=3)
        except Exception as e:
            return "an error occured while downloading the item: " + str(e)

    if not workshop_download_path.exists():
        return None

    items = []
    add_directory(items, workshop_download_path, "%s %s" % (workshop_id, workshop_name))
    atomic = AioZipStream(items, chunksize=32768)

    async with async_open(workshop_zip, mode='wb') as z:
        async for chunk in atomic.stream():
            await z.write(chunk)

    return workshop_zip


def is_empty(item_path: Path):
    return not any(item_path.iterdir())


def add_directory(items, item_path, target_dir):
    if not item_path.exists():
        return

    if item_path.is_file():
        items.append({"file": item_path, "name": target_dir})
    elif item_path.is_dir():
        for item in item_path.iterdir():
            item_name = item.name.replace('\\', '/')
            add_directory(items, item, f"{target_dir}/{item_name}")


async def collection_download(app_id: int, workshop_id: int, validate: bool = True, batch_size: int = 20):
    game_path = steam_path / "steamapps" / "workshop" / "content" / str(app_id)
    title, workshop_matches = await get_collection_items(workshop_id)

    if not workshop_matches or not title:
        return None

    collection_path = game_path / f"{workshop_id} {title}.zip"

    if collection_path.exists() and collection_path.stat().st_size > 0:
        return collection_path

    items = list()

    while workshop_matches:
        batch = workshop_matches[:batch_size]
        workshop_matches = workshop_matches[batch_size:]

        sc = SteamCommand(steam_path)
        for item_id, work_title in batch:
            item_path = game_path / str(item_id)

            add_directory(items, item_path, f"{item_id} {work_title}")

            if item_path.exists() and not is_empty(item_path):
                continue

            sc.workshop_download_item(app_id, item_id, validate)

        if sc.commands:
            await steam.execute(sc, n_tries=3)

    atomic = AioZipStream(items, chunksize=32768)
    async with async_open(collection_path, mode='wb') as z:
        async for chunk in atomic.stream():
            await z.write(chunk)

    return collection_path
