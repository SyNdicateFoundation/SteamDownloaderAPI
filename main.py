from quart import Quart, redirect, request, send_file

from services import (steam_proxied)
from steam import workshop_download, collection_download

app = Quart(__name__)


@app.route("/")
async def index():
    return redirect("/workshop/")


@app.route("/login/home/")
async def login():
    return "We don't allow cross login to steamcommunity.com <a href='/'>Back</a>"


@app.route("/market/")
@app.route("/discussions/")
@app.route("/my/")
@app.route("/id/")
@app.route("/account/")
@app.route("/profiles/")
async def home():
    return "We don't support other pages of steamcommunity.com <a href='/'>Back</a>"


@app.route("/dl-workshop/<int:app_id>/<int:workshop_id>")
async def download_workshop(app_id, workshop_id):
    path = await workshop_download(app_id, workshop_id)
    if path is None:
        return "Workshop not found"
    print(path)
    return await send_file(path)


@app.route("/dl-collection/<int:app_id>/<int:workshop_id>")
async def download_collection(app_id, workshop_id):
    path = await collection_download(app_id, workshop_id)
    if path is None:
        return "Collection not found"

    return await send_file(path)


@app.route("/<path:path>")
async def steam_proxied_wrapper(path):
    return await steam_proxied(path, "steamcommunity.com",
                               headers=dict(request.headers),
                               data=await request.data,
                               method=request.method,
                               params=request.args,
                               cookies=request.cookies,
                               host=request.host)


if __name__ == "__main__":
    app.run(host='0.0.0.0', debug=True)
