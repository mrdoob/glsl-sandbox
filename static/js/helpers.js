
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

		// This is the default code to be loaded
		readURL( '5d00000100a90200000000000000119d88a75623ed1f97decafdbca9f60d5d9aea5c15090d6b99d1bf278123255f7071406b0031147756a95f17500c89f6bb5b982f2996ce0eeec087a302f677faa8ff536d4904bd9e7deb9045d178189362a624333f49df388b60d0d4a174e5d774706548f6322bbca185d84070b143366137ea50788b60c695e83c44b476a2ca324430e11650bb06cc89bc7a19bfd7b93c1a5d382895d9db4865ff7514599a0c6ef3e5e4b232372ee947a42811132b7fa30d635c41c253def1996f6bf984204573e8dc43594c7c9446628b75e9b3ec3c79e14df44b5644785e0ec9ab4ad858a873a001868a7a49ab5d9ad3c7dd40b28e4dc1a4965078b9ce03a8ff5e2fd312cf7031476ad29a4edf64fe07de6907e00913a59f97834d305b9b8c4ffa85b049' );

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

