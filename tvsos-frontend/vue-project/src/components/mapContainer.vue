<script setup>
import { onMounted, onUnmounted, ref, shallowRef, watch } from "vue";
import AMapLoader from "@amap/amap-jsapi-loader";
import { useImformStore, usePoiBoxStore, useMapStore } from "@/stores";
import { getPOIList } from "@/api/poi";
import { storeToRefs } from 'pinia'
const imform = useImformStore();
const poiBox = usePoiBoxStore();

let map = shallowRef(null);
let webglLayerObj = null;

// 存储所有小车实例的Map，方便通过ID查找和更新
const vehiclesMap = shallowRef(new Map());
let AMapInstance = null; // 用于存储AMap全局对象，方便在定时器中使用
let updateInterval = null; // 用于存储定时器ID，以便在组件卸载时清除

const updateFrequencyMs = 1000; // 每秒更新一次数据，即1000毫秒

// ----------------------------------- feature start -----------------------------------------
// 模拟从JSON文件读取车辆数据的函数
let mockTick = 0; // 用于模拟数据变化的计数器，实际上可以去掉
const fullPathColor='#0077ff'
const passedPathColor='#15d100'
const carIcon = "https://a.amap.com/jsapi_demos/static/demo-center-v2/car.png"

//------------------------------------ 模拟数据 -----------------------------------------------
const fetchVehicleData = async () => {
  // 模拟网络延迟
  await new Promise(resolve => setTimeout(resolve, 50));
  mockTick++;

  // 假设的第一辆车的固定路径
  const car1FullPath = [
    [104.065861, 30.6574013], // 天府广场附近
    [104.066500, 30.658000],
    [104.067000, 30.657500],
    [104.067500, 30.658200],
    [104.068000, 30.657800],
    [104.068500, 30.658500],
    [104.069000, 30.657900],
  ];

  // 模拟 car1 沿着路径移动 (循环)
  const pathIndex1 = mockTick % car1FullPath.length;
  const currentPos1 = car1FullPath[pathIndex1];

  // 模拟 car2 的位置随机小范围移动，时不时让其路径为null
  const baseCar2Pos = [104.070000, 30.650000];
  let currentPos2 = [
      baseCar2Pos[0] + Math.sin(mockTick * 0.2) * 0.001,
      baseCar2Pos[1] + Math.cos(mockTick * 0.15) * 0.0005
  ];
  let car2Path = []; // car2 默认不提供完整路径，只有实时位置

  // 模拟 car3 在一段时间后出现，并沿圆周运动
  let car3Data = null;
  if (mockTick > 10) { // 假设10秒后出现
      const baseCar3Pos = [104.061000, 30.660500];
      car3Data = {
          id: 'car3',
          // fullPath 示例，即使实时更新只用 currentPosition，这里也可以显示完整路径
          path: [[104.060000, 30.660000], [104.061000, 30.660500], [104.062000, 30.660000]],
          currentPosition: [
              baseCar3Pos[0] + Math.sin(mockTick * 0.3) * 0.0005,
              baseCar3Pos[1] + Math.cos(mockTick * 0.25) * 0.0003
          ],
          
      };
  }

  const vehicles = [
    {
      id: 'car1',
      path: car1FullPath, // 完整路径
      currentPosition: currentPos1, // 当前位置
    },
    {
      id: 'car2',
      path: car2Path, // car2 的 fullPath 为空，或者有时为null
      currentPosition: currentPos2  
    }
  ];

  // 模拟car2的路径有时为null
  if (mockTick % 10 < 5) { // 每10秒有5秒car2的path为null
      vehicles[1].path = null;
  }

  if (car3Data) {
      vehicles.push(car3Data);
  }

  return vehicles;
};
//-----------------------------------------------------------------------------------------


// 根据新数据更新地图上的车辆，实现平滑移动
const updateVehiclesOnMap = async () => {
    if (!AMapInstance || !map.value) return;

    const newVehicleDataList = await fetchVehicleData();
    const currentCarIds = new Set();

    // 更新或添加车辆
    for (const carData of newVehicleDataList) {
        currentCarIds.add(carData.id);
        let vehicle = vehiclesMap.value.get(carData.id);

        const newPosition = carData.currentPosition;
        if (!newPosition) continue; // 如果没有当前位置，则跳过

        if (!vehicle) {
            // 新增车辆
            vehicle = {
                id: carData.id,
                marker: null,
                fullPolyline: null, // 表示车辆的完整规划路径
                realTimeTrackPolyline: null, // 表示车辆实际实时移动的轨迹
                lastPosition: newPosition, // 记录上一次的位置
                passedAnimationPolyline: null, // 用于moveAlong动画的已走过路径
            };

            vehicle.marker = new AMapInstance.Marker({
                map: map.value,
                position: newPosition, // 初始化位置
                icon: carIcon,
                offset: new AMapInstance.Pixel(-13, -26), // 调整图标偏移量
                autoRotation: true // 开启自动旋转
            });

            // 初始化实时轨迹线 (仅当carData.path不为空时才创建)
            if (carData.path && carData.path.length > 0) {
                 vehicle.realTimeTrackPolyline = new AMapInstance.Polyline({
                    map: map.value,
                    path: [newPosition], // 轨迹线从当前位置开始
                    strokeColor: passedPathColor, // 使用passedPathColor作为实时轨迹线的颜色
                    strokeWeight: 4,
                    strokeStyle: "solid",
                    lineJoin: "round",
                    zIndex: 100 // 确保轨迹线在marker之下但高于基础地图
                });
            }

            // 如果提供了完整路径，则初始化完整路径线
            if (carData.path && carData.path.length > 0) {
                vehicle.fullPolyline = new AMapInstance.Polyline({
                    map: map.value,
                    path: carData.path,
                    showDir: true,
                    strokeColor: fullPathColor,
                    strokeWeight: 6,
                    zIndex: 90 // 完整路径线在实时轨迹线之下
                });
            }
            vehiclesMap.value.set(carData.id, vehicle);

        } else {
            // 更新现有车辆
            const lastPosition = vehicle.lastPosition;

            // 只有当位置发生变化时才进行动画
            if (lastPosition && (lastPosition[0] !== newPosition[0] || lastPosition[1] !== newPosition[1])) {
                // 使用moveAlong实现从上一个点到当前点的平滑移动
                vehicle.marker.moveAlong([lastPosition, newPosition], {
                    duration: updateFrequencyMs, // 动画时长与更新频率一致
                    autoRotation: true,
                });

                // 更新实时轨迹线
                // 如果carData.path存在且有数据，就更新轨迹线
                if (carData.path && carData.path.length > 0) {
                    if (!vehicle.realTimeTrackPolyline) { // 如果不存在则创建
                        vehicle.realTimeTrackPolyline = new AMapInstance.Polyline({
                            map: map.value,
                            path: [newPosition], // 轨迹线从当前位置开始
                            strokeColor: passedPathColor,
                            strokeWeight: 4,
                            strokeStyle: "solid",
                            lineJoin: "round",
                            zIndex: 100
                        });
                    } else { // 存在就更新
                        const currentTrackPath = vehicle.realTimeTrackPolyline.getPath();
                        // 避免重复添加同一个点
                        if (currentTrackPath.length === 0 ||
                            currentTrackPath[currentTrackPath.length - 1][0] !== newPosition[0] ||
                            currentTrackPath[currentTrackPath.length - 1][1] !== newPosition[1]) {
                            currentTrackPath.push(newPosition);
                            vehicle.realTimeTrackPolyline.setPath(currentTrackPath);
                        }
                    }
                } else if (vehicle.realTimeTrackPolyline) { // 如果 carData.path 为空或 null，且轨迹线存在，则移除
                    vehicle.realTimeTrackPolyline.setMap(null);
                    vehicle.realTimeTrackPolyline = null;
                }

            } else if (!lastPosition) { // 如果lastPosition是空的（首次获取到），直接定位
                vehicle.marker.setPosition(newPosition);
                // 此时也检查是否需要初始化或清除实时轨迹线
                if (carData.path && carData.path.length > 0) {
                     if (!vehicle.realTimeTrackPolyline) {
                         vehicle.realTimeTrackPolyline = new AMapInstance.Polyline({
                            map: map.value,
                            path: [newPosition],
                            strokeColor: passedPathColor,
                            strokeWeight: 4,
                            strokeStyle: "solid",
                            lineJoin: "round",
                            zIndex: 100
                        });
                    } else { // 如果已经存在，更新路径为当前位置
                        vehicle.realTimeTrackPolyline.setPath([newPosition]);
                    }
                } else if (vehicle.realTimeTrackPolyline) {
                    vehicle.realTimeTrackPolyline.setMap(null);
                    vehicle.realTimeTrackPolyline = null;
                }
            }

            // 更新lastPosition以供下一次更新使用
            vehicle.lastPosition = newPosition;

            // 如果fullPolyline (完整路径) 是动态变化的，这里需要更新
            if (carData.path && carData.path.length > 0) {
                const currentFullPath = vehicle.fullPolyline ? vehicle.fullPolyline.getPath() : [];
                // 简单的判断路径是否改变，更严谨的应该深比较
                if (!vehicle.fullPolyline || JSON.stringify(currentFullPath) !== JSON.stringify(carData.path)) {
                    if (vehicle.fullPolyline) {
                        vehicle.fullPolyline.setMap(null); // 移除旧路径
                    }
                    vehicle.fullPolyline = new AMapInstance.Polyline({
                        map: map.value,
                        path: carData.path,
                        showDir: true,
                        strokeColor: fullPathColor,
                        strokeWeight: 6,
                        zIndex: 90
                    });
                }
            } else if (vehicle.fullPolyline) {
                // 如果新数据没有路径，但旧车辆有路径，则移除旧路径
                vehicle.fullPolyline.setMap(null);
                vehicle.fullPolyline = null;
            }
        }
    }

    // 移除不再存在的车辆
    for (const [id, vehicle] of vehiclesMap.value.entries()) {
        if (!currentCarIds.has(id)) {
            if (vehicle.marker) {
                vehicle.marker.stopMove(); // 停止可能的动画
                vehicle.marker.setMap(null); // 移除marker
            }
            if (vehicle.fullPolyline) {
                vehicle.fullPolyline.setMap(null); // 移除完整路径
            }
            if (vehicle.realTimeTrackPolyline) {
                vehicle.realTimeTrackPolyline.setMap(null); // 移除实时轨迹线
            }
            if (vehicle.passedAnimationPolyline) {
                vehicle.passedAnimationPolyline.setMap(null); // 移除动画轨迹线
            }
            vehiclesMap.value.delete(id); // 从Map中删除
        }
    }
};

// ------------------------------------ 小车轨迹动画控制函数 (用于基于完整路径的moveAlong) --------------------------------------
const startCarAnimation = () => {
  vehiclesMap.value.forEach(car => {
    if (car.marker && car.fullPolyline && car.fullPolyline.getPath().length > 0) {
      if (!car.passedAnimationPolyline) {
          car.passedAnimationPolyline = new AMapInstance.Polyline({
              map: map.value,
              strokeColor: car.fullPolyline.getOptions().strokeColor, // 继承完整路径颜色
              strokeWeight: 6,
              zIndex: 110 // 比实时轨迹线更高
          });
          // 监听marker的moving事件来实时更新已走过的路径
          car.marker.on('moving', (e) => {
              car.passedAnimationPolyline.setPath(e.passedPath);
          });
      }
      // 启动沿着fullPolyline的moveAlong动画
      car.marker.moveAlong(car.fullPolyline.getPath(), {
        duration: 10000, // 动画总时长，单位毫秒 (根据路径长度调整)
        autoRotation: true,
      });
    }
  });
};

const pauseCarAnimation = () => {
  vehiclesMap.value.forEach(car => {
    if (car.marker) {
      car.marker.pauseMove();
    }
  });
};

const resumeCarAnimation = () => {
  vehiclesMap.value.forEach(car => {
    if (car.marker) {
      car.marker.resumeMove();
    }
  });
};

const stopCarAnimation = () => {
  vehiclesMap.value.forEach(car => {
    if (car.marker) {
      car.marker.stopMove();
      // 重置marker到fullPolyline的起点（如果存在）
      if (car.fullPolyline && car.fullPolyline.getPath().length > 0) {
          car.marker.setPosition(car.fullPolyline.getPath()[0]);
      }
      // 清空已走过的动画路径线
      if (car.passedAnimationPolyline) {
          car.passedAnimationPolyline.setPath([]);
      }
    }
  });
};
//------------------------------------feature end-----------------------------------------

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

// ===== WebGL Layer 工厂函数 (保持不变) =====
function createWebGLLayer(AMap, mapInstance, initialPoiList) {
  let poiList = initialPoiList; // 动态引用数据

  const canvas = document.createElement("canvas");
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

  for (let i = 1; i <= 7; i++) loadTexture(iconMap[i], i, `tex${i}`);
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
  labelContainer.style.zIndex = 1300;
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
      imform.imformShow();
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

const mapStore = useMapStore()
const { center, zoom, blinkingPoi } = storeToRefs(mapStore)
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
      if (!map) return

      // 清除之前的闪烁标记
      if (window.blinkMarker) {
        map.remove(window.blinkMarker)
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
      map.add(marker)

      // 闪烁动画（闪烁6次后自动停止）
      let blinkTimes = 0
      const maxBlinks = 6
      let visible = true

      window.blinkInterval = setInterval(() => {
        visible = !visible
        // 使用正确的方法来控制标记显示/隐藏
        if (visible) {
          marker.show()  // 显示标记
        } else {
          marker.hide()  // 隐藏标记
        blinkTimes++
        }

        if (blinkTimes >= maxBlinks * 2) { // *2 因为每次切换可见性算半次
          clearInterval(window.blinkInterval)
          map.remove(marker)
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
    })
    getPOIList().then((res) => {
      if (res.data?.code === 1 && res.data.data?.length) {
        webglLayerObj = createWebGLLayer(AMap, map.value, res.data.data);
        console.log("初始POI:", res.data.data.length);
      }
    });

    // 启动车辆数据实时更新
    updateVehiclesOnMap(); // 首次加载立即更新一次
    updateInterval = setInterval(updateVehiclesOnMap, updateFrequencyMs); // 每秒更新
  });
});

//-----------------------------------feature end---------------------------------------
onUnmounted(() => {
  // 清除定时器
  if (updateInterval) {
    clearInterval(updateInterval);
    updateInterval = null;
  }

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
  if (map) {
    map.setCenter(newCenter)
  }
})

// 添加监听，当仓库缩放变化时更新地图
watch(zoom, (newZoom) => {
  if (map) {
    map.setZoom(newZoom)
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

        <!-- 轨迹回放控制按钮 (这些按钮现在控制的是车辆沿其完整路径的 moveAlong 动画) -->
        <div class="animation-controls">
          <h4>多车轨迹回放</h4>
          <p style="font-size: 11px; color: #666;">（此功能控制车辆沿着其定义的完整规划路径进行动画，与实时位置更新可能叠加）</p>
          <div class="input-item">
            <el-button @click="startCarAnimation">开始动画</el-button>
            <el-button @click="pauseCarAnimation">暂停动画</el-button>
          </div>
          <div class="input-item">
            <el-button @click="resumeCarAnimation">继续动画</el-button>
            <el-button @click="stopCarAnimation">停止动画</el-button>
          </div>
        </div>
        <hr/> <!-- 分隔符 -->

        <!-- 原有的详细信息 -->
        <div class="detailedInformation">
          编号：{{ poiBox.recentPoi.id }}
        </div><br>
        <div class="detailedInformation">
          名称：{{ poiBox.recentPoi.name }}
        </div><br>
        <div class="detailedInformation">
          类型：{{ poiBox.recentPoiKind }}
        </div><br>
        <div class="detailedInformation">
          状态：{{ poiBox.recentPoiStatus }}
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

#imformBox {
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

/* 轨迹回放控制按钮样式 */
.animation-controls {
  padding: 10px;
  margin-bottom: 10px;
  text-align: center;
  border-bottom: 1px solid #eee; /* 添加分隔线 */
  margin-bottom: 15px;
  padding-bottom: 15px;
}
.animation-controls h4 {
  margin-top: 0;
  margin-bottom: 10px;
  color: #333;
}
.animation-controls .input-item {
  display: flex;
  justify-content: space-around;
  margin-bottom: 10px;
}
.animation-controls .el-button {
  width: 45%;
  border-radius: 5px;
  font-size: 14px;
}
.animation-controls .el-button + .el-button {
  margin-left: 10px;
}
</style>