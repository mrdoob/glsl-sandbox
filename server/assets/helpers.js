
var saveButton;

function initialize_compressor(){
	return null;
}

function load_url_code() {
	if ( window.location.pathname!='/' && window.location.pathname!='/new') {

		load_code(window.location.pathname.substr(1));

	} else {

		code.value = document.getElementById( 'example' ).text;

	}
}

function add_save_button() {
	saveButton = document.createElement( 'button' );
	saveButton.textContent = 'save';
	saveButton.addEventListener( 'click', save, false );
	toolbar.appendChild( saveButton );
}

function set_save_button(visibility) {
	saveButton.style.visibility = visibility;
}

function get_img( width, height ) {
	canvas.width = width;
	canvas.height = height;
	parameters.screenWidth = width;
	parameters.screenHeight = height;

	gl.viewport( 0, 0, width, height );
	createRenderTargets();

	render();

	img=canvas.toDataURL('image/png');

	onWindowResize();

	return img;
}

function save() {
	img=get_img(200, 100);

	data={
		"code": document.getElementById( 'code' ).value,
		"image": img
	}

	$.post(window.location.href,
		JSON.stringify(data),
		function(result) {
			window.location.replace('/'+result);
		}, "text");
}

function load_code(hash) {
	$.get('/item/'+hash, function(result) {
		code.value=result;
		compile();
	});
}

// dummy functions

function setURL(fragment) {
}

