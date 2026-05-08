(() => {
	const VERSION = 10;
	const SIZE = 4 * VERSION + 17;
	const DATA_BLOCK_SIZES = [68, 68, 69, 69];
	const DATA_CODEWORDS = 274;
	const ECC_CODEWORDS = 18;
	const MAX_BYTES = 271;
	const FORMAT_BITS_LOW = 1;
	const MASK_PATTERN = 0;
	const ALIGNMENT_POSITIONS = [6, 28, 50];

	const expTable = new Array(512);
	const logTable = new Array(256);

	let x = 1;
	for (let i = 0; i < 255; i++) {
		expTable[i] = x;
		logTable[x] = i;
		x <<= 1;
		if (x & 0x100) x ^= 0x11d;
	}
	for (let i = 255; i < expTable.length; i++) {
		expTable[i] = expTable[i - 255];
	}

	function render(canvas, text, options) {
		if (!canvas || !canvas.getContext) {
			throw new Error("Canvas is not supported");
		}

		const bytes = utf8Bytes(text);
		if (bytes.length > MAX_BYTES) {
			throw new Error("Link is too long");
		}

		const dataCodewords = makeDataCodewords(bytes);
		const codewords = addEccAndInterleave(dataCodewords);
		const matrix = makeMatrix(codewords);
		drawCanvas(canvas, matrix, options && options.size ? options.size : 260);
	}

	function utf8Bytes(text) {
		if (window.TextEncoder) {
			return Array.from(new TextEncoder().encode(text));
		}
		return Array.from(unescape(encodeURIComponent(text)), (ch) =>
			ch.charCodeAt(0),
		);
	}

	function makeDataCodewords(bytes) {
		const bits = new BitBuffer();
		bits.put(0x4, 4);
		bits.put(bytes.length, 16);
		bytes.forEach((byte) => bits.put(byte, 8));

		const capacityBits = DATA_CODEWORDS * 8;
		const terminatorBits = Math.min(4, capacityBits - bits.length);
		for (let i = 0; i < terminatorBits; i++) bits.putBit(false);
		while (bits.length % 8 !== 0) bits.putBit(false);

		const data = bits.buffer.slice();
		for (let i = 0; data.length < DATA_CODEWORDS; i++) {
			data.push(i % 2 === 0 ? 0xec : 0x11);
		}
		return data;
	}

	function addEccAndInterleave(data) {
		const blocks = [];
		let offset = 0;
		DATA_BLOCK_SIZES.forEach((size) => {
			const block = data.slice(offset, offset + size);
			blocks.push({
				data: block,
				ecc: reedSolomonRemainder(block, ECC_CODEWORDS),
			});
			offset += size;
		});

		const result = [];
		const maxDataSize = Math.max(...DATA_BLOCK_SIZES);
		for (let i = 0; i < maxDataSize; i++) {
			blocks.forEach((block) => {
				if (i < block.data.length) result.push(block.data[i]);
			});
		}
		for (let i = 0; i < ECC_CODEWORDS; i++) {
			blocks.forEach((block) => result.push(block.ecc[i]));
		}
		return result;
	}

	function reedSolomonRemainder(data, degree) {
		const divisor = reedSolomonDivisor(degree);
		const result = data.concat(new Array(degree).fill(0));

		for (let i = 0; i < data.length; i++) {
			const factor = result[i];
			if (factor === 0) continue;
			for (let j = 0; j < divisor.length; j++) {
				result[i + j] ^= gfMultiply(divisor[j], factor);
			}
		}
		return result.slice(data.length);
	}

	function reedSolomonDivisor(degree) {
		let result = [1];
		for (let i = 0; i < degree; i++) {
			result = polynomialMultiply(result, [1, expTable[i]]);
		}
		return result;
	}

	function polynomialMultiply(left, right) {
		const result = new Array(left.length + right.length - 1).fill(0);
		for (let i = 0; i < left.length; i++) {
			for (let j = 0; j < right.length; j++) {
				result[i + j] ^= gfMultiply(left[i], right[j]);
			}
		}
		return result;
	}

	function gfMultiply(left, right) {
		if (left === 0 || right === 0) return 0;
		return expTable[logTable[left] + logTable[right]];
	}

	function makeMatrix(codewords) {
		const matrix = Array.from({ length: SIZE }, () =>
			new Array(SIZE).fill(null),
		);

		drawFinder(matrix, 0, 0);
		drawFinder(matrix, 0, SIZE - 7);
		drawFinder(matrix, SIZE - 7, 0);
		drawAlignmentPatterns(matrix);
		drawTimingPatterns(matrix);
		drawVersionInfo(matrix);
		drawFormatInfo(matrix, MASK_PATTERN);
		setModule(matrix, SIZE - 8, 8, true);
		drawData(matrix, codewords, MASK_PATTERN);
		return matrix;
	}

	function drawFinder(matrix, row, col) {
		for (let dr = -1; dr <= 7; dr++) {
			for (let dc = -1; dc <= 7; dc++) {
				const r = row + dr;
				const c = col + dc;
				if (!inMatrix(r, c)) continue;
				const dark =
					dr >= 0 &&
					dr <= 6 &&
					dc >= 0 &&
					dc <= 6 &&
					(dr === 0 ||
						dr === 6 ||
						dc === 0 ||
						dc === 6 ||
						(dr >= 2 && dr <= 4 && dc >= 2 && dc <= 4));
				setModule(matrix, r, c, dark);
			}
		}
	}

	function drawAlignmentPatterns(matrix) {
		ALIGNMENT_POSITIONS.forEach((row) => {
			ALIGNMENT_POSITIONS.forEach((col) => {
				if (matrix[row][col] !== null) return;
				for (let dr = -2; dr <= 2; dr++) {
					for (let dc = -2; dc <= 2; dc++) {
						const distance = Math.max(Math.abs(dr), Math.abs(dc));
						setModule(matrix, row + dr, col + dc, distance !== 1);
					}
				}
			});
		});
	}

	function drawTimingPatterns(matrix) {
		for (let i = 8; i < SIZE - 8; i++) {
			if (matrix[6][i] === null) setModule(matrix, 6, i, i % 2 === 0);
			if (matrix[i][6] === null) setModule(matrix, i, 6, i % 2 === 0);
		}
	}

	function drawVersionInfo(matrix) {
		const bits = getBchVersion(VERSION);
		for (let i = 0; i < 18; i++) {
			const bit = ((bits >>> i) & 1) !== 0;
			setModule(matrix, Math.floor(i / 3), (i % 3) + SIZE - 11, bit);
			setModule(matrix, (i % 3) + SIZE - 11, Math.floor(i / 3), bit);
		}
	}

	function drawFormatInfo(matrix, maskPattern) {
		const bits = getBchFormat((FORMAT_BITS_LOW << 3) | maskPattern);
		for (let i = 0; i < 15; i++) {
			const bit = ((bits >>> i) & 1) !== 0;
			if (i < 6) {
				setModule(matrix, i, 8, bit);
			} else if (i < 8) {
				setModule(matrix, i + 1, 8, bit);
			} else {
				setModule(matrix, SIZE - 15 + i, 8, bit);
			}

			if (i < 8) {
				setModule(matrix, 8, SIZE - i - 1, bit);
			} else if (i < 9) {
				setModule(matrix, 8, 15 - i, bit);
			} else {
				setModule(matrix, 8, 14 - i, bit);
			}
		}
	}

	function drawData(matrix, codewords, maskPattern) {
		let row = SIZE - 1;
		let direction = -1;
		let bitIndex = 0;
		const totalBits = codewords.length * 8;

		for (let col = SIZE - 1; col > 0; col -= 2) {
			if (col === 6) col--;
			while (true) {
				for (let i = 0; i < 2; i++) {
					const c = col - i;
					if (matrix[row][c] !== null) continue;

					let dark = false;
					if (bitIndex < totalBits) {
						const byte = codewords[Math.floor(bitIndex / 8)];
						dark = ((byte >>> (7 - (bitIndex % 8))) & 1) !== 0;
					}
					if (isMasked(maskPattern, row, c)) dark = !dark;
					matrix[row][c] = dark;
					bitIndex++;
				}

				row += direction;
				if (row < 0 || row >= SIZE) {
					row -= direction;
					direction = -direction;
					break;
				}
			}
		}
	}

	function isMasked(pattern, row, col) {
		if (pattern === 0) return (row + col) % 2 === 0;
		return false;
	}

	function getBchFormat(data) {
		let value = data << 10;
		while (bitLength(value) - bitLength(0x537) >= 0) {
			value ^= 0x537 << (bitLength(value) - bitLength(0x537));
		}
		return ((data << 10) | value) ^ 0x5412;
	}

	function getBchVersion(version) {
		let value = version << 12;
		while (bitLength(value) - bitLength(0x1f25) >= 0) {
			value ^= 0x1f25 << (bitLength(value) - bitLength(0x1f25));
		}
		return (version << 12) | value;
	}

	function bitLength(value) {
		let result = 0;
		while (value !== 0) {
			result++;
			value >>>= 1;
		}
		return result;
	}

	function setModule(matrix, row, col, dark) {
		if (inMatrix(row, col)) matrix[row][col] = dark;
	}

	function inMatrix(row, col) {
		return row >= 0 && row < SIZE && col >= 0 && col < SIZE;
	}

	function drawCanvas(canvas, matrix, requestedSize) {
		const quietZone = 4;
		const modules = matrix.length + quietZone * 2;
		const scale = Math.max(1, Math.floor(requestedSize / modules));
		const size = modules * scale;
		const ratio = window.devicePixelRatio || 1;
		const ctx = canvas.getContext("2d");

		canvas.width = size * ratio;
		canvas.height = size * ratio;
		canvas.style.width = size + "px";
		canvas.style.height = size + "px";

		ctx.setTransform(ratio, 0, 0, ratio, 0, 0);
		ctx.imageSmoothingEnabled = false;
		ctx.fillStyle = "#fff";
		ctx.fillRect(0, 0, size, size);
		ctx.fillStyle = "#111";

		for (let row = 0; row < matrix.length; row++) {
			for (let col = 0; col < matrix.length; col++) {
				if (!matrix[row][col]) continue;
				ctx.fillRect(
					(col + quietZone) * scale,
					(row + quietZone) * scale,
					scale,
					scale,
				);
			}
		}
	}

	function BitBuffer() {
		this.buffer = [];
		this.length = 0;
	}

	BitBuffer.prototype.put = function (value, length) {
		for (let i = length - 1; i >= 0; i--) {
			this.putBit(((value >>> i) & 1) !== 0);
		}
	};

	BitBuffer.prototype.putBit = function (bit) {
		const index = Math.floor(this.length / 8);
		if (this.buffer.length <= index) this.buffer.push(0);
		if (bit) this.buffer[index] |= 0x80 >>> this.length % 8;
		this.length++;
	};

	window.CrossShareQR = { render };
})();
