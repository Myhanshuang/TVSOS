<script setup>
import { onMounted, onUnmounted } from "vue";
import AMapLoader from "@amap/amap-jsapi-loader";
import { useImformStore } from "@/stores";
import { getPOIList } from "@/api/poi";

const imform = useImformStore();
let map = null;
let webglLayerObj = null; // will hold { layer, cleanup }
let animationFrame = null;

// 图标路径（public/images）
const iconMap = {
  1: "/images/加油站.webp",
  2: "/images/加气站.webp",
  3: "/images/其他能源站.webp",
  4: "/images/工厂.webp",
  5: "/images/公司企业.webp",
  6: "/images/购物中心.webp",
  7: "/images/家具建材市场.webp",
};

function createWebGLLayer(AMap, map, poiList) {
  const canvas = document.createElement("canvas");
  canvas.style.position = "absolute";
  canvas.style.left = "0";
  canvas.style.top = "0";
  canvas.style.pointerEvents = "none"; // 不拦截事件
  const gl = canvas.getContext("webgl", { antialias: true, alpha: true });
  if (!gl) {
    console.error("WebGL not supported");
    return null;
  }

  const layer = new AMap.CustomLayer(canvas, {
    zooms: [3, 20],
    alwaysRender: true,
    zIndex: 1200,
  });

  // ---- shaders (保持你原来的着色器，不变) ----
  const vertexShaderSource = `
    attribute vec2 a_Position;
    attribute float a_Type;
    uniform vec2 u_ScreenSize;
    varying float v_Type;
    void main() {
      vec2 clipSpace = (a_Position / u_ScreenSize) * 2.0 - 1.0;
      gl_Position = vec4(clipSpace * vec2(1, -1), 0, 1);
      gl_PointSize = 24.0;
      v_Type = a_Type;
    }
  `;
  const fragmentShaderSource = `
    precision mediump float;
    varying float v_Type;
    uniform sampler2D tex1;
    uniform sampler2D tex2;
    uniform sampler2D tex3;
    uniform sampler2D tex4;
    uniform sampler2D tex5;
    uniform sampler2D tex6;
    uniform sampler2D tex7;
    uniform sampler2D texDefault;
    void main() {
      vec2 uv = gl_PointCoord;
      vec4 color;
      int t = int(v_Type + 0.5);
      if (t == 1) color = texture2D(tex1, uv);
      else if (t == 2) color = texture2D(tex2, uv);
      else if (t == 3) color = texture2D(tex3, uv);
      else if (t == 4) color = texture2D(tex4, uv);
      else if (t == 5) color = texture2D(tex5, uv);
      else if (t == 6) color = texture2D(tex6, uv);
      else if (t == 7) color = texture2D(tex7, uv);
      else color = texture2D(texDefault, uv);
      if (color.a < 0.1) discard;
      gl_FragColor = color;
    }
  `;

  function compileShader(type, source) {
    const s = gl.createShader(type);
    gl.shaderSource(s, source);
    gl.compileShader(s);
    if (!gl.getShaderParameter(s, gl.COMPILE_STATUS)) {
      console.error("Shader compile error:", gl.getShaderInfoLog(s));
    }
    return s;
  }

  const vs = compileShader(gl.VERTEX_SHADER, vertexShaderSource);
  const fs = compileShader(gl.FRAGMENT_SHADER, fragmentShaderSource);
  const program = gl.createProgram();
  gl.attachShader(program, vs);
  gl.attachShader(program, fs);
  gl.linkProgram(program);
  if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
    console.error("Program link error:", gl.getProgramInfoLog(program));
  }
  gl.useProgram(program);

  const a_Position = gl.getAttribLocation(program, "a_Position");
  const a_Type = gl.getAttribLocation(program, "a_Type");
  const u_ScreenSize = gl.getUniformLocation(program, "u_ScreenSize");

  // ---- 记录创建的 GL 资源，方便 cleanup 时释放 ----
  const createdTextures = [];
  const createdBuffers = [];
  const createdShaders = [vs, fs];
  const createdProgram = program;

  // 纹理加载并绑定到指定单元（保留你原有参数）
  function loadTexture(url, unitIndex, uniformName) {
    const tex = gl.createTexture();
    createdTextures.push(tex);
    const img = new Image();
    // img.crossOrigin = "anonymous"; // 若跨域需要启用
    img.src = url;
    img.onload = () => {
      gl.activeTexture(gl["TEXTURE" + unitIndex]);
      gl.bindTexture(gl.TEXTURE_2D, tex);
      // 保留像素存放/绘制相关设置（与原来一致）
      gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, img);
      gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
      gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
      gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR);
      const loc = gl.getUniformLocation(program, uniformName);
      gl.uniform1i(loc, unitIndex);
      // console.log("Loaded texture", url, "-> unit", unitIndex);
    };
    img.onerror = (err) => {
      console.error("Texture load error:", url, err);
    };
    return tex;
  }

  // 绑定图标 (unit 1..7), 0 为默认
  for (let i = 1; i <= 7; i++) {
    loadTexture(iconMap[i], i, `tex${i}`);
  }
  loadTexture(iconMap[1], 0, "texDefault");

  // buffers
  const bufferPos = gl.createBuffer();
  const bufferType = gl.createBuffer();
  createdBuffers.push(bufferPos, bufferType);

  // === 把 layer 加到地图，canvas 会被插入到地图容器中 ===
  layer.setMap(map);

  // === 创建并挂载 label 容器到 map 容器（保证父元素存在） ===
  const mapContainer = map.getContainer ? map.getContainer() : document.getElementById("mapContainer");
  const labelContainer = document.createElement("div");
  labelContainer.style.position = "absolute";
  labelContainer.style.left = "0";
  labelContainer.style.top = "0";
  labelContainer.style.width = "100%";
  labelContainer.style.height = "100%";
  labelContainer.style.pointerEvents = "none";
  labelContainer.style.zIndex = 1300;
  mapContainer.appendChild(labelContainer);

  const hoverLabel = document.createElement("div");
  hoverLabel.style.position = "absolute";
  hoverLabel.style.pointerEvents = "none";
  hoverLabel.style.display = "none";
  hoverLabel.style.padding = "4px 8px";
  hoverLabel.style.background = "rgba(255,255,255,0.95)";
  hoverLabel.style.borderRadius = "6px";
  hoverLabel.style.boxShadow = "0 1px 6px rgba(0,0,0,0.15)";
  hoverLabel.style.fontSize = "12px";
  hoverLabel.style.whiteSpace = "nowrap";
  labelContainer.appendChild(hoverLabel);

  // 像素避让参数
  const cellSize = 20; // 可根据需要调整

  // updateBuffers 保持原逻辑
  function updateBuffers() {
    const positions = [];
    const types = [];
    const occupied = []; // 保存已占用点 screen coords
    const size = map.getSize();
    // 确保 canvas 大小与 map 容器一致（render 调用之前也会设置）
    if (canvas.width !== size.width || canvas.height !== size.height) {
      canvas.width = size.width;
      canvas.height = size.height;
    }

    for (let i = 0; i < poiList.length; i++) {
      const p = poiList[i];
      const pixel = map.lngLatToContainer([p.lon, p.lat]);

      // 屏幕外过滤
      if (pixel.x < -50 || pixel.y < -50 || pixel.x > canvas.width + 50 || pixel.y > canvas.height + 50) continue;

      // 简单像素避让：检查是否和已有保留点冲突
      let tooClose = false;
      for (let k = 0; k < occupied.length; k++) {
        const sp = occupied[k];
        if (Math.abs(pixel.x - sp.x) < cellSize && Math.abs(pixel.y - sp.y) < cellSize) {
          tooClose = true;
          break;
        }
      }
      if (tooClose) continue;

      // 保留并加入缓冲数据
      occupied.push({ x: pixel.x, y: pixel.y });
      positions.push(pixel.x, pixel.y);
      types.push(p.type || 0);
    }

    gl.bindBuffer(gl.ARRAY_BUFFER, bufferPos);
    gl.bufferData(gl.ARRAY_BUFFER, new Float32Array(positions), gl.STATIC_DRAW);

    gl.bindBuffer(gl.ARRAY_BUFFER, bufferType);
    gl.bufferData(gl.ARRAY_BUFFER, new Float32Array(types), gl.STATIC_DRAW);

    return positions.length / 2;
  }

  // renderFrame：**不再在内部 start RAF**（避免重复启动），保持原绘制逻辑
  function renderFrame() {
    const size = map.getSize();
    canvas.width = size.width;
    canvas.height = size.height;
    gl.viewport(0, 0, canvas.width, canvas.height);
    gl.clearColor(0, 0, 0, 0);
    gl.clear(gl.COLOR_BUFFER_BIT);

    const pointCount = updateBuffers();

    gl.uniform2f(u_ScreenSize, canvas.width, canvas.height);

    gl.bindBuffer(gl.ARRAY_BUFFER, bufferPos);
    gl.enableVertexAttribArray(a_Position);
    gl.vertexAttribPointer(a_Position, 2, gl.FLOAT, false, 0, 0);

    gl.bindBuffer(gl.ARRAY_BUFFER, bufferType);
    if (a_Type >= 0) {
      gl.enableVertexAttribArray(a_Type);
      gl.vertexAttribPointer(a_Type, 1, gl.FLOAT, false, 0, 0);
    }

    gl.drawArrays(gl.POINTS, 0, pointCount);

    // **不要在这里调用 requestAnimationFrame**（AMap 会根据 alwaysRender 调用 render）
    // animationFrame = requestAnimationFrame(renderFrame);
  }

  // 把 render 交给 AMap 的 CustomLayer 回调（保持原行为）
  layer.render = renderFrame;

  // 点击 / 移动事件绑定（保持你的原逻辑）
  const clickHandler = (e) => {
    const { pixel } = e;
    let nearest = null;
    let minDist = 20;
    for (let i = 0; i < poiList.length; i++) {
      const p = poiList[i];
      const px = map.lngLatToContainer([p.lon, p.lat]);
      const dx = px.x - pixel.x;
      const dy = px.y - pixel.y;
      const d = Math.sqrt(dx * dx + dy * dy);
      if (d < minDist) {
        minDist = d;
        nearest = p;
      }
    }
    if (nearest) {
      map.setZoomAndCenter(15, [nearest.lon, nearest.lat]);
    }
  };
  map.on("click", clickHandler);

  const moveHandler = (e) => {
    const { pixel } = e;
    let nearest = null;
    let minDist = 14;
    for (let i = 0; i < poiList.length; i++) {
      const p = poiList[i];
      const px = map.lngLatToContainer([p.lon, p.lat]);
      const dx = px.x - pixel.x;
      const dy = px.y - pixel.y;
      const d = Math.sqrt(dx * dx + dy * dy);
      if (d < minDist) {
        minDist = d;
        nearest = p;
      }
    }

    if (nearest) {
      hoverLabel.style.display = "block";
      hoverLabel.innerText = nearest.name;
      hoverLabel.style.left = `${pixel.x + 12}px`;
      hoverLabel.style.top = `${pixel.y - 12}px`;
    } else {
      hoverLabel.style.display = "none";
    }
  };
  map.on("mousemove", moveHandler);

  // 返回 layer 对象和 cleanup —— cleanup 会释放所有 GL 资源、事件、DOM
  return {
    layer,
    cleanup() {
      try {
        // remove map listeners
        map.off("click", clickHandler);
        map.off("mousemove", moveHandler);

        // remove DOM
        if (labelContainer && labelContainer.parentNode) labelContainer.parentNode.removeChild(labelContainer);

        // unset layer
        if (layer) layer.setMap(null);

        // cancel RAF if any leftover (defensive)
        if (typeof animationFrame === "number") cancelAnimationFrame(animationFrame);

        // delete GL buffers
        createdBuffers.forEach((b) => {
          try { gl.deleteBuffer(b); } catch (e) { /* ignore */ }
        });

        // delete textures
        createdTextures.forEach((t) => {
          try { gl.deleteTexture(t); } catch (e) { /* ignore */ }
        });

        // detach program & delete shaders & program
        try {
          if (createdProgram) {
            gl.useProgram(null);
            createdShaders.forEach((s) => { if (s) gl.deleteShader(s); });
            gl.deleteProgram(createdProgram);
          }
        } catch (e) { /* ignore */ }

      } catch (e) {
        console.error("cleanup error:", e);
      }
    },
  };
}
// 生命周期
onMounted(() => {
  window._AMapSecurityConfig = { securityJsCode: "09582d73da9c81d93b134caf4e6f173a" };
  AMapLoader.load({
    key: "84a1985a18fcdb13254b2d85d69885ee",
    version: "2.0",
    plugins: ["AMap.ToolBar", "AMap.Scale"],
  })
    .then((AMap) => {
      map = new AMap.Map("mapContainer", {
        viewMode: "3D",
        zoom: 13,
        center: [104.065861, 30.6574013],
        mapStyle: "amap://styles/whitesmoke",
      });

      map.addControl(new AMap.ToolBar());
      map.addControl(new AMap.Scale());

      getPOIList()
        .then((res) => {
          if (res.data && res.data.code === 1 && res.data.data?.length) {
            webglLayerObj = createWebGLLayer(AMap, map, res.data.data);
            console.log("POI 总数:", res.data.data.length);
          } else {
            console.warn("POI 数据为空");
          }
        })
        .catch((err) => console.error("获取 POI 失败:", err));
    })
    .catch((e) => {
      console.error("AMapLoader.load 失败:", e);
    });
});

onUnmounted(() => {
  if (webglLayerObj && webglLayerObj.cleanup) webglLayerObj.cleanup();
  if (map) {
    map.destroy();
    map = null;
  }
});
</script>



<template>
  <div id="firBorder">
    <div id="mapBox" :class="{ wideMap: !imform.imformIf, shrotMap: imform.imformIf }">
      <div id="mapContainer"></div>
    </div>

    <div id="carImfromBox" :class="{ imformShow: imform.imformIf, imformHide: !imform.imformIf }">
      <div id="imfromBox" :class="{ show: imform.imformIf, hide: !imform.imformIf }">
        这里是小车的信息
      </div>
    </div>
  </div>
</template>

<style scoped>
#firBorder {
  margin: 0;
  padding: 0px 0px;
  width: 100vw;
  height: 92vh;
  display: inline-block;
  text-align: left;
  z-index: 1;
}

#mapBox {
  margin: 0px;
  padding: 0px;
  width: 100%;
  height: 100%;
  display: inline-block;
  position: relative;
  vertical-align: top;
  z-index: 2;
}

#carImfromBox {
  display: inline-block;
  position: absolute;
  right: 2vw;
  top: 13vh;
  vertical-align: top;
  width: 20vw;
  height: 80vh;
  border-radius: 20px;
  background: white;
  box-shadow: 14px 14px 30px #bebebe, -14px -14px 30px #ffffff;
  z-index: 3;
}

#imformBox {
  display: inline-block;
  background-color: aliceblue;
  height: calc(100% - 20px);
  width: calc(100% - 10px);
  margin: 10px 10px 10px 10px;
}

.imformHide {
  margin: 0px;
  padding: 0px;
  width: 0px;
  opacity: 0;
  transform: translateX(200px);
  transition: all 0.4s cubic-bezier(.35, .74, .33, .75) 0.4s;
}

.imformShow {
  margin: 0px 0px 0px 100px;
  border-radius: 25px;
  width: 300px;
  opacity: 1;
  transform: translateX(0px);
  transition: all 0.4s cubic-bezier(.35, .74, .33, .75);
}

.show {
  transform: translateX(0px);
  opacity: 1;
  transition: all 0.4s cubic-bezier(.35, .74, .33, .75) 0.4s;
}

.hide {
  transform: translateX(50px);
  opacity: 0;
  transition: all 0.4s cubic-bezier(.35, .74, .33, .75);
}

#mapContainer {
  width: 100%;
  height: 100%;
}
</style>
