/**
 * WebGL 渲染层 - 用于高性能展示地图上的 POI 点位
 * 该类通过高德地图的 CustomLayer 结合 WebGL 技术，实现在地图上同屏渲染数千个 POI 图标
 */

/** 
 * POI 类型编码规格化 
 * 将后端返回的多种业务类型 ID (1-11) 映射为前端预定义的图标 ID
 */
const mapPoiType = (originalType) => {
    return (originalType >= 1 && originalType <= 11) ? originalType : 11;
};

/** 图标资源映射表 */
const ICON_MAP = {
    1: "/images/加油站.webp", 2: "/images/加气站.webp", 3: "/images/其他能源站.webp",
    4: "/images/工厂.webp", 5: "/images/汽修厂.webp", 6: "/images/物流园.webp",
    7: "/images/火车站.webp", 8: "/images/机场.webp", 9: "/images/购物中心.webp",
    10: "/images/家具建材市场.webp", 11: "/images/公司企业.webp",
};

/** 顶点着色器：处理点的位置转换（经纬度像素 -> WebGL 裁剪空间坐标） */
const VS_SOURCE = `
    attribute vec2 a_Position; // 容器像素坐标
    attribute float a_Type;    // POI 类型
    uniform vec2 u_ScreenSize;
    varying float v_Type;
    void main() {
      // 坐标转换公式：将 [0, width] 映射到 [-1, 1]
      vec2 clipSpace = (a_Position / u_ScreenSize) * 2.0 - 1.0;
      gl_Position = vec4(clipSpace * vec2(1, -1), 0, 1);
      gl_PointSize = 24.0; // 设置图标渲染大小
      v_Type = a_Type;
    }
`;

/** 片元着色器：根据类型选择对应的纹理进行贴图 */
const FS_SOURCE = `
    precision mediump float;
    varying float v_Type;
    uniform sampler2D tex1, tex2, tex3, tex4, tex5, tex6, tex7, tex8, tex9, tex10, texDefault;
    void main() {
      vec2 uv = gl_PointCoord;
      vec4 color;
      int t = int(v_Type + 0.5); // 近似取整判断类型
      if (t == 1) color = texture2D(tex1, uv);
      else if (t == 2) color = texture2D(tex2, uv);
      else if (t == 3) color = texture2D(tex3, uv);
      else if (t == 4) color = texture2D(tex4, uv);
      else if (t == 5) color = texture2D(tex5, uv);
      else if (t == 6) color = texture2D(tex6, uv);
      else if (t == 7) color = texture2D(tex7, uv);
      else if (t == 8) color = texture2D(tex8, uv);
      else if (t == 9) color = texture2D(tex9, uv);
      else if (t == 10) color = texture2D(tex10, uv);
      else color = texture2D(texDefault, uv);
      if (color.a < 0.1) discard; // 丢弃透明边缘
      gl_FragColor = color;
    }
`;

export class WebGLLayer {
    constructor(AMap, mapInstance, options = {}) {
        this.AMap = AMap;
        this.map = mapInstance;
        this.poiList = [];         // 全量原始数据
        this.visiblePoints = [];   // 当前视口内可见的点（用于点击检测）
        this.cellSize = 20;        // 碰撞检测步长（像素），用于稀疏过密的点位

        this.onPoiClick = options.onPoiClick || (() => { });
        this.onEmptyClick = options.onEmptyClick || (() => { });
        this.init();
    }

    /** 初始化 WebGL 环境 */
    init() {
        this.canvas = document.createElement("canvas");
        this.canvas.style.pointerEvents = "none";
        this.gl = this.canvas.getContext("webgl", { antialias: true, alpha: true });

        if (!this.gl) return console.error("WebGL not supported");

        // 创建高德自定义图层
        this.layer = new this.AMap.CustomLayer(this.canvas, {
            zooms: [3, 20],
            alwaysRender: true,
            zIndex: 10,
        });

        this.initShaders();
        this.initTextures();
        this.initLabels();
        this.bindEvents();

        this.layer.render = () => this.renderFrame();
        this.layer.setMap(this.map);
    }

    /** 编译并链接着色器程序 */
    initShaders() {
        const gl = this.gl;
        const compile = (type, src) => {
            const s = gl.createShader(type);
            gl.shaderSource(s, src);
            gl.compileShader(s);
            return s;
        };
        const vs = compile(gl.VERTEX_SHADER, VS_SOURCE);
        const fs = compile(gl.FRAGMENT_SHADER, FS_SOURCE);

        this.program = gl.createProgram();
        gl.attachShader(this.program, vs);
        gl.attachShader(this.program, fs);
        gl.linkProgram(this.program);
        gl.useProgram(this.program);

        this.locations = {
            position: gl.getAttribLocation(this.program, "a_Position"),
            type: gl.getAttribLocation(this.program, "a_Type"),
            screen: gl.getUniformLocation(this.program, "u_ScreenSize")
        };

        this.buffers = {
            pos: gl.createBuffer(),
            type: gl.createBuffer()
        };
        this.shaders = [vs, fs];
    }

    /** 预加载 ICON 图片资源到 GPU 纹理内存 */
    initTextures() {
        this.textures = [];
        const load = (url, index, name) => {
            const tex = this.gl.createTexture();
            this.textures.push(tex);
            const img = new Image();
            img.src = url;
            img.onload = () => {
                const gl = this.gl;
                gl.activeTexture(gl["TEXTURE" + index]);
                gl.bindTexture(gl.TEXTURE_2D, tex);
                gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, img);
                gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
                gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
                gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR);
                gl.uniform1i(gl.getUniformLocation(this.program, name), index);
            };
        };

        for (let i = 1; i <= 10; i++) load(ICON_MAP[i], i, `tex${i}`);
        load(ICON_MAP[1], 0, "texDefault");
    }

    /** 初始化用于展示 POI 名称的浮动 HTML Label 容器 */
    initLabels() {
        const container = this.map.getContainer();
        this.labelContainer = document.createElement("div");
        Object.assign(this.labelContainer.style, {
            position: "absolute", width: "100%", height: "100%",
            pointerEvents: "none", zIndex: 20
        });

        this.hoverLabel = document.createElement("div");
        Object.assign(this.hoverLabel.style, {
            position: "absolute", display: "none", background: "rgba(255,255,255,0.95)",
            padding: "4px 8px", borderRadius: "6px", boxShadow: "0 1px 6px rgba(0,0,0,0.15)",
            fontSize: "12px", whiteSpace: "nowrap"
        });

        this.labelContainer.appendChild(this.hoverLabel);
        container.appendChild(this.labelContainer);
    }

    /** 外部数据入口：设置并标准化 POI 数据列表 */
    updateData(rawList) {
        this.poiList = rawList.map(p => ({ ...p, type: mapPoiType(p.tybe || p.type) }));
        if (this.layer) this.renderFrame();
    }

    /** 更新顶点缓冲区数据，包含视口裁剪和简单抽稀逻辑 */
    updateBuffers() {
        const gl = this.gl;
        const positions = [];
        const types = [];
        const occupied = []; // 用于像素级的碰撞剔除
        this.visiblePoints = [];

        const size = this.map.getSize();
        this.canvas.width = size.width;
        this.canvas.height = size.height;

        for (const p of this.poiList) {
            const pixel = this.map.lngLatToContainer([p.lon, p.lat]);
            // 1. 视口裁剪：只处理画布及其周边 40px 内的点
            if (pixel.x < -40 || pixel.y < -40 || pixel.x > size.width + 40 || pixel.y > size.height + 40) continue;

            // 2. 抽稀：防止图标在层级较低时堆叠导致视觉混乱
            const tooClose = occupied.some(sp => Math.abs(pixel.x - sp.x) < this.cellSize && Math.abs(pixel.y - sp.y) < this.cellSize);
            if (tooClose) continue;

            occupied.push({ x: pixel.x, y: pixel.y });
            positions.push(pixel.x, pixel.y);
            types.push(p.type || 0);
            this.visiblePoints.push({ ...p, pixel });
        }

        gl.bindBuffer(gl.ARRAY_BUFFER, this.buffers.pos);
        gl.bufferData(gl.ARRAY_BUFFER, new Float32Array(positions), gl.STATIC_DRAW);
        gl.bindBuffer(gl.ARRAY_BUFFER, this.buffers.type);
        gl.bufferData(gl.ARRAY_BUFFER, new Float32Array(types), gl.STATIC_DRAW);

        return positions.length / 2;
    }

    /** 高德地图触发的重绘帧方法 */
    renderFrame() {
        const gl = this.gl;
        const size = this.map.getSize();
        gl.viewport(0, 0, size.width, size.height);
        gl.clear(gl.COLOR_BUFFER_BIT);

        const count = this.updateBuffers();
        gl.uniform2f(this.locations.screen, size.width, size.height);

        gl.bindBuffer(gl.ARRAY_BUFFER, this.buffers.pos);
        gl.enableVertexAttribArray(this.locations.position);
        gl.vertexAttribPointer(this.locations.position, 2, gl.FLOAT, false, 0, 0);

        gl.bindBuffer(gl.ARRAY_BUFFER, this.buffers.type);
        gl.enableVertexAttribArray(this.locations.type);
        gl.vertexAttribPointer(this.locations.type, 1, gl.FLOAT, false, 0, 0);

        gl.drawArrays(gl.POINTS, 0, count);
    }

    /** 处理地图交互事件（点击、移动检测） */
    bindEvents() {
        this._onClick = (e) => {
            const nearest = this.findNearest(e.pixel, 20); // 20px 点击半径
            if (nearest) this.onPoiClick(nearest);
            else this.onEmptyClick();
        };

        this._onMouseMove = (e) => {
            const nearest = this.findNearest(e.pixel, 14); // 悬浮悬浮提示灵敏度
            if (nearest) {
                this.hoverLabel.style.display = "block";
                this.hoverLabel.innerText = nearest.name;
                this.hoverLabel.style.left = `${e.pixel.x + 12}px`;
                this.hoverLabel.style.top = `${e.pixel.y - 12}px`;
            } else {
                this.hoverLabel.style.display = "none";
            }
        };

        this.map.on("click", this._onClick);
        this.map.on("mousemove", this._onMouseMove);
    }

    /** 空间搜索：查找鼠标位置附近最近的 POI 点 */
    findNearest(pixel, threshold) {
        let nearest = null;
        let minDist = threshold;
        for (const p of this.visiblePoints) {
            const d = Math.sqrt((p.pixel.x - pixel.x) ** 2 + (p.pixel.y - pixel.y) ** 2);
            if (d < minDist) {
                minDist = d;
                nearest = p;
            }
        }
        return nearest;
    }

    /** 销毁组件：清理 WebGL 显存资源、DOM 节点及高德地图监听器 */
    cleanup() {
        if (this.map) {
            this.map.off("click", this._onClick);
            this.map.off("mousemove", this._onMouseMove);
        }
        if (this.labelContainer?.parentNode) {
            this.labelContainer.parentNode.removeChild(this.labelContainer);
        }
        if (this.layer) this.layer.setMap(null);

        const gl = this.gl;
        Object.values(this.buffers).forEach(b => gl.deleteBuffer(b));
        this.textures.forEach(t => gl.deleteTexture(t));
        this.shaders.forEach(s => gl.deleteShader(s));
        gl.deleteProgram(this.program);
    }
}