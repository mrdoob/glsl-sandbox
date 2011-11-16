
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

		readURL( '5d00000100980200000000000000119a48c65ab5aec1f910f780dfdfe473e599a211a90304ab6a650a0bdc710e60d9ef6827f7e37c460aba047c4de9e20bce74f0e6773fe3b4b7d379f6f885aacf345507100f3a9c00b35ece337c99a5b1914781cf1261e20c852069d976e19e0626035accf277b6d605f6f79b5b829acddc05289378c5e94ed5e728c24b0c22e42ddd138eaafc87372557f72d2dd04c4538fde32958381dcc055e8bb8c995f6f131916a68f6a9eae6d314121e0fbcfc26aed27e4a9b352caf72ef1b2d94e7a0c30bb73bdceac95fd45a10ae5d0cba4fb744a5d815c78fe091f2be7dae03592fa89dc80524475f0d296359c067472f2efcf9f2695185e5a5d2a5cdf31d8ea098e48054863d3489cf72c148e9ac8fbb401c229d9e08e9b8ff39ced000' );

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

		code.value = text;
		compile();

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

function add_save_button() {
}

