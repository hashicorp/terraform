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
	},

	// Determine color fill?
	update: function(engine){},

	draw: function(ctx, scale){
		ctx.moveTo(
			this.a.pos.x * scale >> 0,
			this.a.pos.y * scale >> 0
		);
		ctx.lineTo(
			this.b.pos.x * scale >> 0,
			this.b.pos.y * scale >> 0
		);
		ctx.lineTo(
			this.c.pos.x * scale >> 0,
			this.c.pos.y * scale >> 0
		);
		ctx.lineTo(
			this.a.pos.x * scale >> 0,
			this.a.pos.y * scale >> 0
		);
	}

};

})(window.Engine, window.Vector);
