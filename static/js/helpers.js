
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

		readURL( '5d00000100b70200000000000000381c88cdaf8125d4569ed1e6e6c09c2fe72b7d489ad9d27ce026c1d90b38e6a986e7c482f98001c7d016ca8db7da32debe67fc602659f4e96ae150d75ea26ae8e8f4056e0b845d0814a2acee3a47ec45af66fb0405385eaedd50db968497c1cbbd79f0634d5534182b2093986dae5ac7e840eb137f39f5b27efbb488f4e85faa6f4c942182673e75dca44f7ffbd8c25c5a2763cb750b9b4e14b9fd5c15ca81ed7ef88a4f77a114cd2d7f675b6a05b467bd164f4058dd3250e6fcdc9d7c195dccdc63b304e1f8c7aaccb1edb7992aa1341e96aa6e8b1ca044e70207be752de1b41e4e07843ab895b2a1995b97085afff70f5eff9d2b630a4c5e43e7ce6f977616d57c97935342e48d0ee9ad3462804ae5608b6872b9749632bf8514fea3ebf9d64b10deb3ff041395050deffbf038e3' );

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

