(function(){

CubeDraw = Base.extend({

	$cube: null,
	ROWS: 4,
	PADDING: 64, // 44 pixel base square + 20 padding
	previousRowTop: null,
	previousRowLeft: null,
	lastCube: null,

	constructor: function(){
		this.$cube = $('.cube');
		this.$cubes = $('#cubes');

		this.lastCube = this.$cube;
		this.previousRowLeft = parseInt(this.lastCube.css('left'), 10)
		this.previousRowTop = parseInt(this.lastCube.css('top'), 10);

		this.addEventListeners();
	},

	addEventListeners: function(){
		var angle = this.getRadiansForAngle(30);
		var sin = Math.sin(angle) * this.PADDING;
		var cos = Math.cos(angle) * this.PADDING;

		//sett up our parent columns
		for(var i = 0; i < this.ROWS; i++){
			var cube = this.lastCube.clone();

			cube.css({ top: this.previousRowTop - sin, left: this.previousRowLeft - cos});
			this.$cubes.prepend(cube);
			this.lastCube = cube;
			this.previousRowLeft = parseInt(this.lastCube.css('left'), 10)
			this.previousRowTop = parseInt(this.lastCube.css('top'), 10)
		}

		//use the parent cubes as starting point for rows
		var $allParentCubes = $('.cube');
		var angle = this.getRadiansForAngle(150);
		var sin = Math.sin(angle) * this.PADDING;
		var cos = Math.cos(angle) * this.PADDING;

		for(var j = this.ROWS; j > -1 ; j--){
			var baseCube = $($allParentCubes[j]);

			this.previousRowLeft = parseInt(baseCube.css('left'), 10)
			this.previousRowTop = parseInt(baseCube.css('top'), 10)

			for(var n = 0; n < this.ROWS; n++){
				var cube = baseCube.clone();
				cube.css({ top: this.previousRowTop - sin, left: this.previousRowLeft - cos});

				this.$cubes.prepend(cube);

				this.lastCube = cube;
				this.previousRowLeft = parseInt(this.lastCube.css('left'), 10)
				this.previousRowTop = parseInt(this.lastCube.css('top'), 10)
			}
		}

		var $all = $('.cube');
		for(var c = 0; c < $all.length; c++){
			(function(index){
				setTimeout(function(){
					var $theCube = $($all[index]);
					$theCube.addClass('in')
				}, 100*c)
			})(c)
		}
	},

	getRadiansForAngle: function(angle) {
		return angle * (Math.PI/180);
	}

});

window.CubeDraw = CubeDraw;

})();
