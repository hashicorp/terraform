(function(
	Engine,
	Vector
){

Engine.Polygon.Puller = function(a, b, c, color, simple){
	this.a = a;
	this.b = b;
	this.c = c;

	this.strokeStyle = '#ffffff';
};

Engine.Polygon.Puller.prototype = {

	checkChasing: function(){
		if (
			this.a._chasing === true &&
			this.b._chasing === true &&
			this.c._chasing === true
		) {
			return true;
		}
		return false;
	}

};

})(window.Engine, window.Vector);
