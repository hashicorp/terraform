(function(String){

if (String.prototype.substitute) {
	return;
}

String.prototype.substitute = function(object, regexp){
	return String(this).replace(regexp || (/\\?\{([^{}]+)\}/g), function(match, name){
		if (match.charAt(0) == '\\') return match.slice(1);
		return (object[name] !== null) ? object[name] : '';
	});
};

})(String);
