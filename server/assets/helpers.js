
var saveButton;
var effect_owner=false;
var original_code='';

function initialize_compressor(){
	return null;
}

function initialize_helper() {
	if ( !localStorage.getItem('glslsandbox_user') )
		localStorage.setItem('glslsandbox_user', generate_user_id());
}

function generate_user_id() {
	return (Math.random()*0x10000000|0).toString(16);
}

function get_user_id() {
	return localStorage.getItem('glslsandbox_user');
}

function am_i_owner() {
	return (effect_owner==false || effect_owner==get_user_id());
}

function load_url_code() {
	if ( window.location.pathname!='/' && window.location.pathname!='/new') {

		load_code(window.location.pathname.substr(1));

	} else {

		code.value = document.getElementById( 'example' ).text;
		original_code = document.getElementById( 'example' ).text;

	}
}

function add_save_button() {
	saveButton = document.createElement( 'button' );
	saveButton.textContent = 'save';
	saveButton.addEventListener( 'click', save, false );
	toolbar.appendChild( saveButton );
}

function set_save_button(visibility) {
	if(original_code==code.value)
		saveButton.style.visibility = 'hidden';
	else
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
		"image": img,
		"user": get_user_id()
	}

	loc='/new';

	if(am_i_owner())
		loc=window.location.href;

	$.post(loc,
		JSON.stringify(data),
		function(result) {
			window.location.replace('/'+result);
		}, "text");
}

function load_code(hash) {
	$.getJSON('/item/'+hash, function(result) {
		code.value=result['code'];
		original_code=result['code'];
		effect_owner=result['user'];

		if(am_i_owner())
			saveButton.textContent = 'save';
		else
			saveButton.textContent = 'fork';

		compile();
	});
}

// dummy functions

function setURL(fragment) {
}

