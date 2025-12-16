<script setup>
import { onMounted, onUnmounted, ref, shallowRef, watch } from "vue";
import AMapLoader from "@amap/amap-jsapi-loader";
import { useMapAnimationStore,useImformStore, usePoiBoxStore, useMapStore } from "@/stores";
import { getPOIList } from "@/api/poi";
import { storeToRefs } from 'pinia';

const imform = useImformStore();
const poiBox = usePoiBoxStore();
const mapStore = useMapStore();
const { center, zoom, blinkingPoi } = storeToRefs(mapStore);
const { currentInfoType, recentVehicle } = storeToRefs(imform);
const { recentPoi } = storeToRefs(poiBox);

let map = shallowRef(null);
let webglLayerObj = null;

// 存储所有小车实例的Map，方便通过ID查找和更新
const vehiclesMap = shallowRef(new Map());
let AMapInstance = null; // 用于存储AMap全局对象，方便在定时器中使用

// 全局车辆默认图标、路径颜色定义
const DEFAULT_VEHICLE_ICON = "https://a.amap.com/jsapi_demos/static/demo-center-v2/car.png";
const VEHICLE_FULL_PATH_COLOR = "#28F";    // 车辆完整规划路径颜色
const VEHICLE_PASSED_PATH_COLOR = "#AF5"; // 车辆实时运动轨迹颜色 (Passed Path Color)
const updateFrequencyMs = 2000; // 更新一次数据的时间

// 控制轮询状态的变量，用于控制按钮的禁用状态
const mapAnimationStore = useMapAnimationStore();

// ----------------------------------- 动画控制函数：现在是包装器 --------------------------------------
// 准备传递给服务函数的选项对象
const getServiceOptions = () => ({
    AMapInstance: AMapInstance,
    map: map.value,
    vehiclesMap: vehiclesMap,
    updateFrequencyMs: updateFrequencyMs,
    DEFAULT_VEHICLE_ICON: DEFAULT_VEHICLE_ICON,
    VEHICLE_FULL_PATH_COLOR: VEHICLE_FULL_PATH_COLOR,
    VEHICLE_PASSED_PATH_COLOR: VEHICLE_PASSED_PATH_COLOR,
    imformStore: imform
});

// 新增：车辆状态映射函数
const getVehicleStatusText = (status) => {
  const statusMap = {
    1: '空闲',
    2: '接单行驶',
    3: '装货',
    4: '运货行驶',
    5: '卸货中',
    6: '停留等待',
    7: '加油',
    8: '维修'
  };
  return statusMap[status] || '未知状态'; // 如果没有匹配的状态，显示'未知状态'
};

const getVehicleCategoryText = (category) => {
  const statusMap = {
    1: '平板货车',
    2: '高护栏货车',
    3: '厢式货车',
    4: '冷链运输车',
    5: '危化品运输车',
  };
  return statusMap[category] || '未知类型'; // 如果没有匹配的状态，显示'未知类型'
};

//------------------------------------feature end-----------------------------------------



// 【新增】用于在信息面板中显示车辆实时位置的 ref
const displayPosition = ref(null);
// 【新增】用于存储位置更新定时器的变量
let positionInterval = null;

// 【新增】使用 watch 监听当前选中的车辆
watch(recentVehicle, (newVehicle) => {
  // 1. 首先清除上一个定时器，防止内存泄漏或冲突
  if (positionInterval) {
    clearInterval(positionInterval);
    positionInterval = null;
  }

  // 2. 如果有新选中的车辆，并且当前显示的就是车辆信息
  if (newVehicle && currentInfoType.value === 'vehicle') {
    // 从 map 中找到这个车辆的完整实例（包含 marker）
    const vehicle = vehiclesMap.value.get(newVehicle.id);
    
    if (vehicle && vehicle.marker) {
      const updateDisplayPosition = () => {
        const currentPos = vehicle.marker.getPosition();
        if (currentPos) {
          // AMap 的 getPosition() 返回的是一个对象，我们需要经纬度数组
          displayPosition.value = [currentPos.getLng(), currentPos.getLat()];
        }
      };

      // 立即执行一次，确保点击后立刻显示正确位置
      updateDisplayPosition();
      
      // 启动一个定时器，高频更新位置（例如每 100 毫秒）
      positionInterval = setInterval(updateDisplayPosition, 100);
    }
  } else {
    // 3. 如果没有选中车辆（例如关闭了信息面板），则清空位置信息
    displayPosition.value = null;
  }
}, { deep: true });






// 图标路径（public/images） (这个部分是POI点的图标，与车辆无关，所以保持不变)
const iconMap = {
    1: "/images/加油站.webp",
    2: "/images/加气站.webp",
    3: "/images/其他能源站.webp",
    4: "/images/工厂.webp",
    5: "/images/汽修厂.webp",
    6: "/images/物流园.webp",
    7: "/images/火车站.webp",
    8: "/images/机场.webp",
    9: "/images/购物中心.webp",
    10: "/images/家具建材市场.webp",
};

// **新增：POI类型映射函数**
const mapPoiType = (originalType) => {
    if (originalType >= 1 && originalType <= 3) {
        return originalType;
    } else if (originalType >= 4 && originalType <= 9) {
        return 4; // 工厂
    } else if (originalType >= 10 && originalType <= 15) {
        return 5; // 汽修厂
    } else if (originalType >= 16 && originalType <= 18) {
        return 6; // 物流园
    } else if (originalType === 19) {
        return 7; // 火车站
    } else if (originalType === 20) {
        return 8; // 机场
    } else if (originalType >= 21 && originalType <= 22) {
        return 9; // 购物中心
    } else if (originalType >= 23 && originalType <= 25) {
        return 10; // 家具建材市场
    }
    return 1; // 默认值，以防出现未知类型
};

// ===== WebGL Layer 工厂函数 (保持不变) =====
function createWebGLLayer(AMap, mapInstance, initialPoiList) {
    let poiList = initialPoiList; // 动态引用数据

    const canvas = document.createElement("canvas");
    canvas.style.pointerEvents = "none"; // 允许鼠标事件穿透 WebGL canvas
    const gl = canvas.getContext("webgl", { antialias: true, alpha: true });
    if (!gl) {
        console.error("WebGL not supported");
        return null;
    }

    const layer = new AMap.CustomLayer(canvas, {
        zooms: [3, 20],
        alwaysRender: true,
        zIndex: 10,
    });

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
    uniform sampler2D tex8;
    uniform sampler2D tex9;
    uniform sampler2D tex10;

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
      else if (t == 8) color = texture2D(tex8, uv);
      else if (t == 9) color = texture2D(tex9, uv);
      else if (t == 10) color = texture2D(tex10, uv);

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
    gl.useProgram(program);

    const a_Position = gl.getAttribLocation(program, "a_Position");
    const a_Type = gl.getAttribLocation(program, "a_Type");
    const u_ScreenSize = gl.getUniformLocation(program, "u_ScreenSize");

    const createdTextures = [];
    const createdBuffers = [];
    const createdShaders = [vs, fs];
    const createdProgram = program;

    function loadTexture(url, unitIndex, uniformName) {
        const tex = gl.createTexture();
        createdTextures.push(tex);
        const img = new Image();
        img.src = url;
        img.onload = () => {
            gl.activeTexture(gl["TEXTURE" + unitIndex]);
            gl.bindTexture(gl.TEXTURE_2D, tex);
            gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, img);
            gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
            gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
            gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR);
            gl.uniform1i(gl.getUniformLocation(program, uniformName), unitIndex);
        };
    }

    // **修改：只加载 iconMap 中存在的 10 种类型的纹理**
    for (let i = 1; i <= 10; i++) loadTexture(iconMap[i], i, `tex${i}`);
    loadTexture(iconMap[1], 0, "texDefault");

    const bufferPos = gl.createBuffer();
    const bufferType = gl.createBuffer();
    createdBuffers.push(bufferPos, bufferType);

    layer.setMap(mapInstance);

    const mapContainer = mapInstance.getContainer();
    const labelContainer = document.createElement("div");
    labelContainer.style.position = "absolute";
    labelContainer.style.width = "100%";
    labelContainer.style.height = "100%";
    labelContainer.style.pointerEvents = "none";
    labelContainer.style.zIndex = 20;
    mapContainer.appendChild(labelContainer);

    const hoverLabel = document.createElement("div");
    hoverLabel.style.position = "absolute";
    hoverLabel.style.display = "none";
    hoverLabel.style.background = "rgba(255,255,255,0.95)";
    hoverLabel.style.padding = "4px 8px";
    hoverLabel.style.borderRadius = "6px";
    hoverLabel.style.boxShadow = "0 1px 6px rgba(0,0,0,0.15)";
    hoverLabel.style.fontSize = "12px";
    hoverLabel.style.whiteSpace = "nowrap";
    labelContainer.appendChild(hoverLabel);

    const cellSize = 20;

    let visiblePoints = [];

    function updateBuffers() {
        const positions = [];
        const types = [];
        const occupied = [];
        visiblePoints = [];

        const size = mapInstance.getSize();
        canvas.width = size.width;
        canvas.height = size.height;

        for (let i = 0; i < poiList.length; i++) {
            const p = poiList[i];
            const pixel = mapInstance.lngLatToContainer([p.lon, p.lat]);
            if (pixel.x < -50 || pixel.y < -50 || pixel.x > size.width + 50 || pixel.y > size.height + 50)
                continue;

            let tooClose = false;
            for (let sp of occupied) {
                if (Math.abs(pixel.x - sp.x) < cellSize && Math.abs(pixel.y - sp.y) < cellSize) {
                    tooClose = true;
                    break;
                }
            }
            if (tooClose) continue;

            occupied.push({ x: pixel.x, y: pixel.y });
            positions.push(pixel.x, pixel.y);
            types.push(p.type || 0);
            visiblePoints.push({ ...p, pixel });
        }

        gl.bindBuffer(gl.ARRAY_BUFFER, bufferPos);
        gl.bufferData(gl.ARRAY_BUFFER, new Float32Array(positions), gl.STATIC_DRAW);
        gl.bindBuffer(gl.ARRAY_BUFFER, bufferType);
        gl.bufferData(gl.ARRAY_BUFFER, new Float32Array(types), gl.STATIC_DRAW);
        return positions.length / 2;
    }

    function renderFrame() {
        const size = mapInstance.getSize();
        canvas.width = size.width;
        canvas.height = size.height;
        gl.viewport(0, 0, size.width, size.height);
        gl.clearColor(0, 0, 0, 0);
        gl.clear(gl.COLOR_BUFFER_BIT);

        const pointCount = updateBuffers();
        gl.uniform2f(u_ScreenSize, size.width, size.height);
        gl.bindBuffer(gl.ARRAY_BUFFER, bufferPos);
        gl.enableVertexAttribArray(a_Position);
        gl.vertexAttribPointer(a_Position, 2, gl.FLOAT, false, 0, 0);
        gl.bindBuffer(gl.ARRAY_BUFFER, bufferType);
        gl.enableVertexAttribArray(a_Type);
        gl.vertexAttribPointer(a_Type, 1, gl.FLOAT, false, 0, 0);
        gl.drawArrays(gl.POINTS, 0, pointCount);
    }

    layer.render = renderFrame;

    const clickHandler = (e) => {
        const { pixel } = e;
        let nearest = null;
        let minDist = 20;
        for (const p of visiblePoints) {
            const dx = p.pixel.x - pixel.x;
            const dy = p.pixel.y - pixel.y;
            const d = Math.sqrt(dx * dx + dy * dy);
            if (d < minDist) {
                minDist = d;
                nearest = p;
            }
        }
        if (nearest) {
            poiBox.recentPoiChange(nearest);
            console.log(poiBox.recentPoi);
            mapInstance.setZoomAndCenter(15, [nearest.lon, nearest.lat]);
            imform.imformShow('poi');
        } else {
          imform.imformHide();//如果没点到任何点，隐藏详细信息栏
        }

    };

    const moveHandler = (e) => {
        const { pixel } = e;
        let nearest = null;
        let minDist = 14;
        for (const p of visiblePoints) {
            const dx = p.pixel.x - pixel.x;
            const dy = p.pixel.y - pixel.y;
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

    mapInstance.on("click", clickHandler);
    mapInstance.on("mousemove", moveHandler);

    return {
        layer,
        updatePoiList(newList) {
            poiList = newList || [];
            layer.render();
        },
        cleanup() {
            if (mapInstance) {
                mapInstance.off("click", clickHandler);
                mapInstance.off("mousemove", moveHandler);
            }
            if (labelContainer?.parentNode) labelContainer.parentNode.removeChild(labelContainer);
            if (layer) layer.setMap(null);
            createdBuffers.forEach((b) => gl.deleteBuffer(b));
            createdTextures.forEach((t) => gl.deleteTexture(t));
            gl.useProgram(null);
            createdShaders.forEach((s) => gl.deleteShader(s));
            gl.deleteProgram(createdProgram);
        },
    };
}

// ===== 生命周期 =====
onMounted(() => {
    
    window._AMapSecurityConfig = { securityJsCode: "09582d73da9c81d93b134caf4e6f173a" };
    AMapLoader.load({
        key: "84a1985a18fcdb13254b2d85d69885ee",
        version: "2.0",
        plugins: ["AMap.ToolBar", "AMap.Scale", "AMap.MoveAnimation"],
    }).then((AMap) => {
        AMapInstance = AMap; // 存储 AMap 实例
        map.value = new AMap.Map("mapContainer", {
            viewMode: "3D",
            center: center.value,
            zoom: zoom.value,
            mapStyle: "amap://styles/whitesmoke",
        });

        map.value.addControl(new AMap.ToolBar());
        map.value.addControl(new AMap.Scale());

        const startBlinkAnimation = (poi) => {
            if (!map.value) return

            // 清除之前的闪烁标记
            if (window.blinkMarker) {
                map.value.remove(window.blinkMarker)
                clearInterval(window.blinkInterval)
            }

            // 创建闪烁标记
            const marker = new AMap.Marker({
                position: [poi.lon, poi.lat],
                content: createBlinkContent(poi.type),
                zIndex: 9999,
                offset: new AMap.Pixel(-12, -12),

            })
            window.blinkMarker = marker
            map.value.add(marker)

            // 闪烁动画（闪烁6次后自动停止）
            let blinkTimes = 0
            let visible = true

            window.blinkInterval = setInterval(() => {
                visible = !visible
                if (visible) {
                    marker.show()
                } else {
                    marker.hide()
                }
                blinkTimes++

                if (blinkTimes >= 6 * 2) {
                    clearInterval(window.blinkInterval)
                    map.value.remove(marker)
                    window.blinkMarker = null
                    mapStore.setBlinkingPoi(null)
                }
            }, 300)
        }
        // 创建闪烁标记内容
        const createBlinkContent = (type) => {
            const iconUrl = iconMap[type] || iconMap[1]
            return `
                <div style="
                    width: 24px;
                    height: 24px;
                    background-image: url('${iconUrl}');
                    background-size: cover;
                    border: 2px solid #ff0000;
                    border-radius: 50%;
                    box-shadow: 0 0 10px rgba(255,0,0,0.8);
                "></div>
            `
        }

        // 监听闪烁POI变化
        watch(blinkingPoi, (newPoi) => {
            if (newPoi) {
                startBlinkAnimation(newPoi)
            }
        }, { deep: true }) // 添加 deep: true 以确保复杂对象的变化也能被监测

        getPOIList().then((res) => {
          if (res.data?.code === 1 && res.data.data?.length) {
            const formattedPoiList = res.data.data.map(poi => ({
                ...poi,
                // **修改：应用类型映射函数**
                type: mapPoiType(poi.tybe)
            }));
          webglLayerObj = createWebGLLayer(AMap, map.value, formattedPoiList);
          console.log("初始POI:", formattedPoiList.length);
          }
        });
        // 将地图上下文信息传递给 Store
        mapAnimationStore.setPollingOptions(getServiceOptions());
    });
});

onUnmounted(() => {
    // 调用服务中的停止轮询函数
    if (positionInterval) clearInterval(positionInterval);
    stopPolling();
    mapAnimationStore.globalStopPolling();
    // 清理WebGL图层
    if (webglLayerObj?.cleanup) webglLayerObj.cleanup();

    // 移除所有车辆相关的地图元素
    vehiclesMap.value.forEach(car => {
        if (car.marker) {
            car.marker.stopMove();
            car.marker.setMap(null); // 移除marker
        }
        if (car.fullPolyline) car.fullPolyline.setMap(null); // 移除完整路径
        if (car.realTimeTrackPolyline) car.realTimeTrackPolyline.setMap(null); // 移除实时轨迹线
        if (car.passedAnimationPolyline) car.passedAnimationPolyline.setMap(null); // 移除动画轨迹线
    });
    vehiclesMap.value.clear(); // 清空Map

    // 销毁地图实例
    if (map.value) map.value.destroy();
});

// 添加监听，当仓库中心点变化时更新地图
watch(center, (newCenter) => {
    if (map.value && newCenter) {
        map.value.setCenter(newCenter)
    }
}, { deep: true })

// 添加监听，当仓库缩放变化时更新地图
watch(zoom, (newZoom) => {
    if (map.value && newZoom !== undefined) {
        map.value.setZoom(newZoom)
    }
})
</script>

<template>
    <div id="firBorder">
        <div id="mapBox" :class="{ wideMap: !imform.imformIf, shrotMap: imform.imformIf }">
            <div id="mapContainer"></div>
        </div>
        <div id="carImfromBox" :class="{ imformShow: imform.imformIf, imformHide: !imform.imformIf }">
            <div id="imfromBox" :class="{ show: imform.imformIf, hide: !imform.imformIf }">
                <el-button class="imformOut" @click="imform.imformHide">×</el-button>
                <!-- 根据 currentInfoType 动态渲染详细信息 -->
                <div v-if="currentInfoType === 'poi' && recentPoi">
                    <h3>POI 详细信息</h3>
                    <div class="detailedInformation">编号：{{ recentPoi.id }}</div><br>
                    <div class="detailedInformation">名称：{{ recentPoi.name }}</div><br>
                    <!-- 假设 poiBox.recentPoiKind 和 poiBox.recentPoiStatus 是根据 recentPoi 实时计算或获取的 -->
                    <div class="detailedInformation">类型：{{ poiBox.recentPoiKind }}</div><br>
                    <div class="detailedInformation">状态：{{ poiBox.recentPoiStatus }}</div>
                </div>
                <div v-else-if="currentInfoType === 'vehicle' && recentVehicle">
                    <h3>车辆详细信息</h3>
                    <div class="detailedInformation">ID：{{ recentVehicle.id }}</div><br>
                    <div class="detailedInformation">车牌号：{{ recentVehicle.license }}</div><br>
                    <div class="detailedInformation">车辆类型：{{ getVehicleCategoryText(recentVehicle.categoryId) }}</div><br>
                    <div class="detailedInformation">位置：{{ displayPosition?.[0]?.toFixed(5) }}, {{ displayPosition?.[1]?.toFixed(5) }}</div><br>
                    <div class="detailedInformation">速度：{{ recentVehicle.speed }} km/h</div><br> 
                    <div class="detailedInformation">当前状态：{{ getVehicleStatusText(recentVehicle.status) }}</div><br>
                    <div class="detailedInformation">运输距离：{{ recentVehicle.distance==null?"NaN": recentVehicle.distance.toFixed(2) }}（km）</div><br>
                    <div class="detailedInformation">预计到达时间：{{ recentVehicle.duration==null?"NaN":recentVehicle.duration.toFixed(2) }}（小时）</div><br>
                    <div class="detailedInformation">最后更新时间：{{ recentVehicle.updateTime }}</div><br>
                    <!-- 你需要根据实际的 vehicle 数据结构添加更多字段 -->
                </div>
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

#imfromBox {
    display: inline-block;
    background-color: aliceblue;
    height: calc(100% - 20px);
    width: calc(100% - 10px);
    margin: 10px 10px 10px 10px;
    position: relative;
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

.detailedInformation {
    display: inline-block;
    margin: 15px 10px 5px 15px;
}

.imformOut {
    display: inline-block;
    margin: 0px;
    padding: 0px;
    width: 30px;
    height: 30px;
    position: absolute;
    top: 10px;
    right: 10px;
    font-size: large;
    border-radius: 50%;
    border: 0px;
    transition: all 0.5s;
}

.imformOut:hover {
    color: rgb(255, 14, 14);
    background-color: white;
}

</style>