'use strict';

function Audio(numBins, cutoff, smooth, scale) {

	numBins = numBins || 3;
	smooth = smooth || 0.4;

	function tick() {
		if (this.meyda) {
			var features = this.meyda.get()
			if (features) {
				var reducer = (accumulator, currentValue) => accumulator + currentValue;

				var spacing = Math.floor(features.loudness.specific.length / this.bins.length);
				this.prevBins = this.bins.slice(0);

				this.bins = this.bins.map((bin, index) =>
					features.loudness.specific.slice(index * spacing, (index + 1) * spacing).reduce(reducer) / spacing
				).map((bin, index) =>
					bin * (1.0 - smooth) + this.prevBins[index] * smooth);

				this.fft = this.bins;
			}
		}
	}

	function init() {
		this.bins = Array(numBins).fill(0)
		this.prevBins = Array(numBins).fill(0)
		this.fft = Array(numBins).fill(0)

		window.navigator.mediaDevices.getUserMedia({ video: false, audio: true })
			.then((stream) => {
				var context = new AudioContext()
				var audio_stream = context.createMediaStreamSource(stream)

				this.meyda = Meyda.createMeydaAnalyzer({
					audioContext: context,
					source: audio_stream,
					featureExtractors: [
						'loudness',
					]
				})
			})
			.catch((err) => console.log('ERROR', err))

		return this;
	}

	return { init: init, tick: tick }.init();
}
