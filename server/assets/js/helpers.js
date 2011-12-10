
var saveButton, forkButton, parentButton;
var effect_owner=false;
var original_code='';
var original_version='';

function initialize_compressor(){
	return null;
}

function initialize_helper() {
	window.onhashchange = function() { load_url_code(); };

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
	if ( window.location.hash!='') {

		load_code(window.location.hash.substr(1));

	} else {

		code.setValue(document.getElementById( 'example' ).text);
		original_code = document.getElementById( 'example' ).text;

	}
}

function add_server_buttons() {
	saveButton = document.createElement( 'button' );
	saveButton.textContent = 'save';
	saveButton.addEventListener( 'click', save, false );
	toolbar.appendChild( saveButton );

	parentButton = document.createElement( 'a' );
	parentButton.textContent = 'parent';
	parentButton.href = original_version;
	toolbar.appendChild( parentButton );

	set_parent_button('visible');
}

function set_save_button(visibility) {
	if(original_code==code.getValue())
		saveButton.style.visibility = 'hidden';
	else
		saveButton.style.visibility = visibility;
}

function set_parent_button(visibility) {
	if(original_version=='')
		parentButton.style.visibility = 'hidden';
	else
		parentButton.style.visibility = visibility;
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
		"code": code.getValue(),
		"image": img,
		"user": get_user_id()
	}

	loc='/e';

	if(am_i_owner())
		data["code_id"]=window.location.hash.substr(1);
	else {
		data["parent"]=window.location.hash.substr(1);
	}

	$.post(loc,
		JSON.stringify(data),
		function(result) {
			window.location.replace('/e#'+result);
			load_url_code();
		}, "text");
}

function load_code(hash) {
	$.getJSON('/item/'+hash, function(result) {
		code.setValue(result['code']);
		original_code=result['code'];

		if(result['parent']) {
			original_version=result['parent'];
			parentButton.href = original_version;
			set_parent_button('visible');
		} else {
			original_version='';
			parentButton.href = '/';
			set_parent_button('hidden');
		}

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

