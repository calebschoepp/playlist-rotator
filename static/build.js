function build(playlistID) {
  var url = window.location.protocol + "//" + window.location.host;
  url = url + "/playlist/" + playlistID + "/build";
  console.log(url);
  const Http = new XMLHttpRequest();
  Http.open("POST", url);
  Http.send();

  Http.onreadystatechange = (e) => {
    if (Http.readyState !== Http.DONE) {
      return;
    }
    console.dir(Http);
  };
}
