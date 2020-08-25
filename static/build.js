function buildPlaylist(playlistID) {
  var tagElem = document.querySelector("#build-tag-" + playlistID);
  console.log(tagElem);
  tagElem.setAttribute("src", "/static/building_pill.svg");

  var url = window.location.protocol + "//" + window.location.host;
  url = url + "/playlist/" + playlistID + "/build";
  const Http = new XMLHttpRequest();
  Http.open("POST", url);
  Http.send();

  Http.onreadystatechange = (e) => {
    if (Http.readyState !== Http.DONE) {
      return;
    }
    if (Http.status != 202) {
      tagElem.setAttribute("src", "/static/failed_pill.svg");
    }
  };
}

function deletePlaylist(playlistID, elemID) {
  var url = window.location.protocol + "//" + window.location.host;
  url = url + "/playlist/" + playlistID + "/delete";
  console.log(url);
  console.log(elemID);
  const Http = new XMLHttpRequest();
  Http.open("DELETE", url);
  Http.send();

  Http.onreadystatechange = (e) => {
    if (Http.readyState !== Http.DONE) {
      return;
    }
    var element = document.querySelector("#" + elemID);
    element.parentNode.removeChild(element);
    return;
  };
}
