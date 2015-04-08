
function initialize_compressor() {
	compressor=new LZMA( "js/lzma_worker.js" );
	return compressor;
}

function initialize_helper() {
}

function load_url_code() {
	if ( window.location.hash ) {

		var hash = window.location.hash.substr( 1 );
		var version = hash.substr( 0, 2 );

		if ( version == 'A/' ) {

			// LZMA

			readURL( hash.substr( 2 ) );

		} else {

			// Basic format

			code.value = decodeURIComponent( hash );

		}

	} else {

		readURL( '5d00000100860200000000000000381c88cdaf8125d4569ed1e6e6c09c2fe72b7d489ad9d27ce026c849f505dd720ff335d10a9aae020d6c5ae383c1b48113253ebdf2f3ce134f3820f3c0c51fbc05ad011e6760c37538ef2510e11f33cb7080000de4c452d3fc136ece677b67088acbd792a9b5ce5df0c751f3b4524f01d0d87382a85e78d7c74a94532bf2216c7659751c13f005aa3d330478c93ec81671986980aa7fe7a7b9972d62db986e6a7786ea36ae9d56eb16d980a18602322ed6531c3174e62dc3d3ada99b0b48be05ce9e0b5dc31c6d8baf62ed660490bf621322abf89d13028ed7794077b5a072711d7aed1cb7d89d92a1a81866aad6582ae8930795aa5c0a31646d43343ed71bb6fce5230efd39c66e' );

	}
}

function setURL( shaderString ) {

	compressor.compress( shaderString, 1, function( bytes ) {

		var hex = convertBytesToHex( bytes );
		window.location.replace( '#A/' + hex );

	},
	dummyFunction );

}

function readURL( hash ) {

	var bytes = convertHexToBytes( hash );

	compressor.decompress( bytes, function( text ) {

		compileOnChangeCode = false;  // Prevent compile timer start
		code.setValue(text);
		compile();
		compileOnChangeCode = true;

	},
	dummyFunction );

}

function convertHexToBytes( text ) {

	var tmpHex, array = [];

	for ( var i = 0; i < text.length; i += 2 ) {

		tmpHex = text.substring( i, i + 2 );
		array.push( parseInt( tmpHex, 16 ) );

	}

	return array;

}

function convertBytesToHex( byteArray ) {

	var tmpHex, hex = "";

	for ( var i = 0, il = byteArray.length; i < il; i ++ ) {

		if ( byteArray[ i ] < 0 ) {

			byteArray[ i ] = byteArray[ i ] + 256;

		}

		tmpHex = byteArray[ i ].toString( 16 );

		// add leading zero

		if ( tmpHex.length == 1 ) tmpHex = "0" + tmpHex;

		hex += tmpHex;

	}

	return hex;

}

// dummy functions for saveButton
function set_save_button(visibility) {
}

function set_parent_button(visibility) {
}

function add_server_buttons() {
}

