from re import compile

from cache import AsyncTTL
from httpx import AsyncClient
from quart.wrappers.response import DataBody, Response

host_fixer = compile(r"^http(s?)://[\w.:]+/")
info = compile(rb"onclick=\"(?:SubscribeItem|SubscribeCollection|SubscribeCollectionItem)"
               rb"\(\s+'(\d+)',\s+'(\d+)'\s+\);\"")
item_rex = compile(rb"(<a\s+onclick=\"SubscribeCollectionItem\(\s+'(\d+)',\s+'(\d+)'\s+\);"
                   rb"\"\s+id=\"SubscribeItemBtn\d+\"\s+class=\"[^\"]*\"\s*>[\s\S]*?</a>)")


@AsyncTTL(time_to_live=60, maxsize=40 << 10)
async def replace_headers(headers: dict, target_host: str) -> dict:
    if "Referer" in headers:
        headers["Referer"] = host_fixer.sub("https://%s/" % target_host, headers["Referer"])

    if "Origin" in headers:
        headers["Origin"] = host_fixer.sub("https://%s/" % target_host, headers["Origin"])

    if "Host" in headers:
        headers["Host"] = "steamcommunity.com"

    if "x-frame-options" in headers:
        del headers["x-frame-options"]

    if "content-security-policy" in headers:
        del headers["content-security-policy"]

    if "content-encoding" in headers:
        del headers["content-encoding"]

    if "access-control-allow-origin" in headers:
        del headers["access-control-allow-origin"]

    if "location" in headers:
        del headers["location"]

    return headers


@AsyncTTL(time_to_live=60, maxsize=40 << 10)
async def fix_body(content, host: str, target_host: str) -> bytes:
    target_host = target_host.encode()
    content = content.replace(b"https://%s/" % target_host, b"/")
    content = content.replace(b"https://%s" % target_host, b"")
    content = content.replace(target_host, host.encode())
    return content


def try_to_add_download_button(data_body):
    if b"<a onclick=\"SubscribeItem(" in data_body:
        index = data_body.find(b"<a onclick=\"SubscribeItem(")

        if index != -1:
            while chr(data_body[index - 1]).isspace():
                index -= 1

            data_info = info.search(data_body[index:])
            if not data_info:
                return data_body

            workshop_id, app_id = data_info.groups()
            if not workshop_id or not app_id:
                return data_body

            index -= 5

            data_body = data_body[:index] + (b"<div><a href=\"/dl-workshop/%s/%s\" class=\"btn_darkred_white_innerfade "
                                             b"btn_border_2px btn_medium\" style=\"position: relative\"> <div "
                                             b"class=\"followIcon\"></div> <span class=\"subscribeText\"> "
                                             b"<div>Download</div> </span> </a> </div>" % (
                                                 app_id, workshop_id)) + data_body[index:]
    elif b"<a class=\"general_btn subscribe\" onclick=\"SubscribeCollection(":
        index = data_body.find(b"<a class=\"general_btn subscribe\" onclick=\"SubscribeCollection(")

        if index != -1:
            data_info = info.search(data_body[index:])
            if not data_info:
                return data_body

            workshop_id, app_id = data_info.groups()
            if not workshop_id or not app_id:
                return data_body

            index -= 5

            data_body = data_body[:index] + (b"<a class=\"general_btn subscribe\" "
                                             b"style=\"background: #640000; color: white;\" "
                                             b"href=\"/dl-collection/%s/%s\">"
                                             b"<div class=\"followIcon\"></div> <span "
                                             b"class=\"subscribeText\">Download Collection</span> </a>" % (
                                                 app_id, workshop_id)) + data_body[index:]

            data_body = item_rex.sub(lambda m: b"<a style=\"background: #640000; color: white;\""
                                               b" href=\"/dl-workshop/%s/%s\""
                                               b" class=\"general_btn subscribe\""
                                               b"><div class=\"followIcon\"></div></a> %s" % (
                m.group(3), m.group(2), m.group(1)), data_body)

    return data_body


async def steam_proxied(path: str, target_host: str, headers: dict, host: str, **kwargs) -> Response:
    headers = await replace_headers(headers, target_host)

    headers["User-Agent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0"

    async with AsyncClient() as client:
        response = await client.request(url="https://%s/%s" % (target_host, path),
                                        headers=headers,
                                        **kwargs)

        data_body = response.content
        data_headers = dict(response.headers.items())

        data_body = await fix_body(data_body, host, "community.akamai.steamstatic.com")
        data_headers = await replace_headers(data_headers, "community.akamai.steamstatic.com")

        data_body = await fix_body(data_body, host, "steamcommunity.com")
        data_headers = await replace_headers(data_headers, "steamcommunity.com")

        data_body = try_to_add_download_button(data_body)

        data_headers["content-length"] = len(data_body)

        return Response(response=DataBody(data_body),
                        headers=data_headers,
                        status=response.status_code)
