function addNewSourceInput() {
  // Get source input info from select
  var selectID = "sourceOptions";
  var potentialSources = document.getElementById(selectID).options;
  var idx = potentialSources.selectedIndex;
  var id = potentialSources[idx].id;
  if (id === "") {
    id = "LIKEDID";
  }
  var name = potentialSources[idx].innerText;
  name = encodeURIComponent(name);
  var type = potentialSources[idx].classList[0];

  // Make request to server to get html
  var url =
    window.location.protocol +
    "//" +
    window.location.host +
    window.location.pathname;
  url = url + "/source/type/" + type;
  url = url + "/name/" + name;
  url = url + "/id/" + id;
  const Http = new XMLHttpRequest();
  Http.open("GET", url);
  Http.send();

  Http.onreadystatechange = (e) => {
    if (Http.readyState !== Http.DONE) {
      return;
    }
    var fragment = createFragment(Http.responseText);
    var inputID = "#source-input-holder";
    var inputHolder = document.querySelector(inputID);
    inputHolder.insertBefore(fragment, inputHolder.lastElementChild);
  };
}

function createFragment(htmlStr) {
  var frag = document.createDocumentFragment(),
    temp = document.createElement("div");
  temp.innerHTML = htmlStr;
  while (temp.firstChild) {
    frag.appendChild(temp.firstChild);
  }
  return frag;
}

function deleteSourceInput(id) {
  var element = document.querySelector("#" + id);
  element.parentNode.removeChild(element);
  return;
}
