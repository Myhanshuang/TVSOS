import { getVehiclesData } from '@/api/vehicle';
import { useVehicleStore } from '@/stores';

let vehiclesSocket = null;
const pathSockets = new Map();
let reconnectTimer = null;
let isRealtimeRunning = false;
let runtimeOptions = null;

const MOVE_DURATION_MS = 9000;
const DEFAULT_STATIC_VEHICLE_ANGLE = 90;
const MIN_MOVE_DISTANCE_M = 1;
const MAX_JUMP_DISTANCE_M = 15000; // 15km to account for high-speed simulation
const MIN_MOVE_DURATION_MS = 400;
const MAX_MOVE_DURATION_MS = 4500;
const SEGMENT_PAUSE_MS = 10;

const toFiniteNumber = (value) => {
  const n = Number(value);
  return Number.isFinite(n) ? n : null;
};

const parsePointToLngLat = (p) => {
  if (!p) return null;

  if (typeof p.getLng === 'function' && typeof p.getLat === 'function') {
    const lon = toFiniteNumber(p.getLng());
    const lat = toFiniteNumber(p.getLat());
    return lon !== null && lat !== null ? [lon, lat] : null;
  }

  if (typeof p === 'string') {
    const parts = p.split(',');
    if (parts.length >= 2) {
      const lon = toFiniteNumber(parts[0]);
      const lat = toFiniteNumber(parts[1]);
      return lon !== null && lat !== null ? [lon, lat] : null;
    }
    return null;
  }

  if (Array.isArray(p) && p.length >= 2) {
    const lon = toFiniteNumber(p[0]);
    const lat = toFiniteNumber(p[1]);
    return lon !== null && lat !== null ? [lon, lat] : null;
  }

  const lon = toFiniteNumber(p.lon ?? p.Lon ?? p.lng ?? p.Lng ?? p.longitude ?? p.Longitude);
  const lat = toFiniteNumber(p.lat ?? p.Lat ?? p.latitude ?? p.Latitude);
  if (lon === null || lat === null) return null;
  return [lon, lat];
};

const isValidLngLat = (point) => {
  if (!Array.isArray(point) || point.length < 2) return false;
  const lon = toFiniteNumber(point[0]);
  const lat = toFiniteNumber(point[1]);
  if (lon === null || lat === null) return false;
  return lon >= -180 && lon <= 180 && lat >= -90 && lat <= 90;
};

const sanitizePolylinePath = (rawPoints) => {
  if (!Array.isArray(rawPoints)) return [];
  const points = [];
  for (const raw of rawPoints) {
    const parsed = parsePointToLngLat(raw);
    if (!isValidLngLat(parsed)) continue;
    const prev = points[points.length - 1];
    if (prev && prev[0] === parsed[0] && prev[1] === parsed[1]) continue;
    points.push(parsed);
  }
  return points;
};

const safeSetPolylinePath = (polyline, rawPoints) => {
  if (!polyline) return false;
  const points = sanitizePolylinePath(rawPoints);
  if (points.length < 2) {
    polyline.hide();
    return false;
  }
  polyline.setPath(points);
  polyline.show();
  return true;
};

const buildPassedPathByProgress = (vehicle, progressIndex, fallbackPoint) => {
  const points = Array.isArray(vehicle?.fullPathPoints) ? vehicle.fullPathPoints : [];
  if (points.length === 0) {
    return sanitizePolylinePath(fallbackPoint ? [fallbackPoint] : []);
  }

  const idx = Number(progressIndex);
  if (!Number.isFinite(idx)) {
    return sanitizePolylinePath([points[0]]);
  }

  const safeIndex = Math.max(0, Math.min(points.length - 1, Math.floor(idx)));
  const history = points.slice(0, safeIndex + 1);
  return sanitizePolylinePath(history);
};

const calcMoveDurationMs = (vehicle, currentMessageTs, distanceMeters) => {
  // Use timestamp delta to align with backend simulation cycle time, acting as a failsafe
  if (vehicle && vehicle.lastVehicleMsgTs > 0 && currentMessageTs > vehicle.lastVehicleMsgTs) {
    const delta = currentMessageTs - vehicle.lastVehicleMsgTs;
    // Failsafe: if delta is within a reasonable simulation interval (e.g., 0.1s to 30s)
    if (delta >= 100 && delta <= 30000) {
      // Use 95% of delta to ensure animation finishes just before next point arrives
      return Math.max(MIN_MOVE_DURATION_MS, delta * 0.95);
    }
  }
  
  // Fallback: backend VehicleSimulateMovingGap is 10s, if no delta, use ~9.5s
  if (Number.isFinite(distanceMeters) && distanceMeters > 0) {
    const duration = distanceMeters * 25; // legacy distance-based estimate
    // but scale it closer to the 10s cycle
    return Math.max(MIN_MOVE_DURATION_MS, Math.min(9500, duration));
  }
  return 9500;
};

const getMarkerLngLat = (marker) => {
  if (!marker || typeof marker.getPosition !== 'function') return null;
  return parsePointToLngLat(marker.getPosition());
};

const syncVehiclePassedPathWithPoint = (vehicle, currentPoint) => {
  if (!vehicle?.passedPolyline) return;
  const progressIdx = Number(vehicle.lastProgressIndex);
  if (!Number.isFinite(progressIdx) || progressIdx < 0) return;
  if (!isValidLngLat(currentPoint)) return;

  const passedPath = buildPassedPathByProgress(vehicle, progressIdx, currentPoint);
  vehicle.passedPathHistory = passedPath;
  safeSetPolylinePath(vehicle.passedPolyline, passedPath);
};

const stopVehicleAnimation = (vehicle) => {
  if (!vehicle) return;
  if (vehicle.animationFrameId) {
    cancelAnimationFrame(vehicle.animationFrameId);
    vehicle.animationFrameId = null;
  }
  if (vehicle.segmentPauseTimer) {
    clearTimeout(vehicle.segmentPauseTimer);
    vehicle.segmentPauseTimer = null;
  }
  if (vehicle.onMovingHandler && vehicle.marker) {
    vehicle.marker.off('moving', vehicle.onMovingHandler);
    vehicle.onMovingHandler = null;
  }
};

const animateMarkerSegment = (vehicle, marker, startPosition, endPosition, durationMs, pauseMs = SEGMENT_PAUSE_MS) => {
  if (!vehicle || !marker || !isValidLngLat(startPosition) || !isValidLngLat(endPosition)) {
    return;
  }

  stopVehicleAnimation(vehicle);

  // Use AMap's built-in moveTo for strict straight line animation between cached points
  const safeDuration = Math.max(1, durationMs || MIN_MOVE_DURATION_MS);
  
  if (typeof marker.moveTo === 'function') {
    marker.moveTo(endPosition, {
      duration: safeDuration,
      autoRotation: false
    });
    
    // To sync passed path we might need to listen to moving event
    vehicle.onMovingHandler = () => {
      const pos = marker.getPosition();
      if (pos) {
        syncVehiclePassedPathWithPoint(vehicle, [pos.getLng(), pos.getLat()]);
      }
    };
    marker.on('moving', vehicle.onMovingHandler);
    
    vehicle.segmentPauseTimer = setTimeout(() => {
      if (vehicle.onMovingHandler) {
        marker.off('moving', vehicle.onMovingHandler);
        vehicle.onMovingHandler = null;
      }
      vehicle.segmentPauseTimer = null;
      syncVehiclePassedPathWithPoint(vehicle, endPosition);
    }, safeDuration + 50);
  } else {
    // Fallback to manual requestAnimationFrame
    const sx = startPosition[0];
    const sy = startPosition[1];
    const ex = endPosition[0];
    const ey = endPosition[1];
    const startedAt = performance.now();

    const step = (now) => {
      const elapsed = now - startedAt;
      const t = Math.min(1, elapsed / safeDuration);
      const lon = sx + (ex - sx) * t;
      const lat = sy + (ey - sy) * t;
      const framePoint = [lon, lat];
      marker.setPosition(framePoint);
      syncVehiclePassedPathWithPoint(vehicle, framePoint);

      if (t < 1) {
        vehicle.animationFrameId = requestAnimationFrame(step);
        return;
      }

      vehicle.animationFrameId = null;
      marker.setPosition(endPosition);
      syncVehiclePassedPathWithPoint(vehicle, endPosition);
      if (pauseMs > 0) {
        vehicle.segmentPauseTimer = setTimeout(() => {
          vehicle.segmentPauseTimer = null;
        }, pauseMs);
      }
    };

    vehicle.animationFrameId = requestAnimationFrame(step);
  }
};

const getVehicleColor = (id) => {
  const colors = [
    '#FF5733', '#33FF57', '#3357FF', '#FF33A1', '#33FFF5',
    '#F5FF33', '#FF8C33', '#8C33FF', '#33FF8C', '#FF3333'
  ];
  return colors[id % colors.length];
};

const getWsBase = () => {
  const envBase = import.meta.env.VITE_WS_BASE;
  if (envBase) {
    return envBase.replace(/\/$/, '');
  }
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${window.location.host}/ws`;
};

const normalizeVehicle = (raw) => {
  const id = Number(raw?.id ?? 0);
  const lon = toFiniteNumber(raw?.lon);
  const lat = toFiniteNumber(raw?.lat);
  const status = toFiniteNumber(raw?.status);
  const speed = toFiniteNumber(raw?.speed);
  const categoryId = toFiniteNumber(raw?.categoryId ?? raw?.tybe);

  return {
    ...raw,
    id,
    lon,
    lat,
    status,
    speed,
    categoryId,
    angle: toFiniteNumber(raw?.angle),
    license: raw?.license,
    currentPosition: [lon, lat]
  };
};

const mergeVehicleData = (prevData, incomingData) => {
  const merged = {
    ...(prevData || {}),
    ...(incomingData || {})
  };

  merged.id = Number(merged?.id ?? 0);
  merged.lon = toFiniteNumber(merged?.lon);
  merged.lat = toFiniteNumber(merged?.lat);
  merged.status = toFiniteNumber(merged?.status) ?? 0;
  merged.speed = toFiniteNumber(merged?.speed) ?? 0;
  merged.categoryId = toFiniteNumber(merged?.categoryId ?? merged?.tybe) ?? 1;
  merged.angle = toFiniteNumber(merged?.angle) ?? DEFAULT_STATIC_VEHICLE_ANGLE;
  merged.license = merged?.license || `车辆-${merged.id}`;
  merged.currentPosition = [merged.lon, merged.lat];

  return merged;
};

const ensurePathSocket = (vehicleId, options) => {
  if (pathSockets.has(vehicleId)) return;

  const ws = new WebSocket(`${getWsBase()}/paths/${vehicleId}`);
  pathSockets.set(vehicleId, ws);

  ws.onmessage = (evt) => {
    let msg;
    try {
      msg = JSON.parse(evt.data);
    } catch {
      return;
    }
    const payload = msg?.payload || {};
    const event = msg?.event;
    const ts = normalizeMessageTs(msg?.ts);
    const vehicle = options.vehiclesMap.value.get(vehicleId);
    if (!vehicle) return;
    if (ts < (vehicle.lastPathMsgTs || 0)) return;
    vehicle.lastPathMsgTs = ts;

    if (event === 'full_path') {
      const points = sanitizePolylinePath(payload.points);
      const routeVersion = payload.routeVersion ?? payload.RouteVersion ?? null;
      const snapshotProgress = Number(payload.progress);

      if (points.length < 2) {
        vehicle.fullPolyline?.setMap(null);
        vehicle.passedPolyline?.setMap(null);
        vehicle.fullPolyline = null;
        vehicle.passedPolyline = null;
        vehicle.fullPathPoints = null;
        return;
      }

      vehicle.fullPathPoints = points;
      vehicle.activeRouteVersion = routeVersion;
      vehicle.activeShipmentId = Number(payload.shipmentId ?? payload.ShipmentId ?? 0) || null;
      vehicle.passedPathHistory = [];
      vehicle.lastProgressIndex = Number.isFinite(snapshotProgress) ? snapshotProgress : -1;
      if (!vehicle.fullPolyline) {
        vehicle.fullPolyline = new options.AMapInstance.Polyline({
          map: options.map,
          path: points,
          strokeColor: '#999999',
          strokeOpacity: 0.3,
          strokeWeight: 6,
          zIndex: 80
        });
      } else {
        safeSetPolylinePath(vehicle.fullPolyline, points);
      }

      const dynamicColor = getVehicleColor(vehicleId);
      if (!vehicle.passedPolyline) {
        vehicle.passedPolyline = new options.AMapInstance.Polyline({
          map: options.map,
          strokeColor: dynamicColor,
          strokeOpacity: 0.9,
          strokeWeight: 6,
          zIndex: 81
        });
        vehicle.passedPolyline.hide();
      } else {
        vehicle.passedPolyline.setOptions({ strokeColor: dynamicColor });
        vehicle.passedPolyline.hide();
      }

      if (Number.isFinite(snapshotProgress) && snapshotProgress >= 0) {
        const anchorPoint = vehicle.lastReportedPosition || getMarkerLngLat(vehicle.marker);
        if (isValidLngLat(anchorPoint)) {
          syncVehiclePassedPathWithPoint(vehicle, anchorPoint);
        }
      }
    }

    if (event === 'progress') {
      const shipmentId = Number(payload.shipmentId ?? payload.ShipmentId ?? 0) || null;
      const routeVersion = payload.routeVersion ?? payload.RouteVersion ?? null;
      if (vehicle.activeShipmentId && shipmentId && vehicle.activeShipmentId !== shipmentId) {
        return;
      }
      if (vehicle.activeRouteVersion && routeVersion && vehicle.activeRouteVersion !== routeVersion) {
        return;
      }

      const progressIdx = Number(payload.progress);
      if (Number.isFinite(progressIdx) && progressIdx < (vehicle.lastProgressIndex ?? -1)) {
        return;
      }
      if (Number.isFinite(progressIdx)) {
        vehicle.lastProgressIndex = progressIdx;
      }

      const anchorPoint = getMarkerLngLat(vehicle.marker) || vehicle.lastReportedPosition;
      if (isValidLngLat(anchorPoint)) {
        syncVehiclePassedPathWithPoint(vehicle, anchorPoint);
      }
    }

    if (event === 'clear') {
      const routeVersion = payload.routeVersion ?? payload.RouteVersion ?? null;
      if (vehicle.activeRouteVersion && routeVersion && vehicle.activeRouteVersion !== routeVersion) {
        return;
      }
      vehicle.fullPolyline?.setMap(null);
      vehicle.passedPolyline?.setMap(null);
      vehicle.fullPolyline = null;
      vehicle.passedPolyline = null;
      vehicle.fullPathPoints = null;
      vehicle.activeRouteVersion = null;
      vehicle.activeShipmentId = null;
      vehicle.passedPathHistory = [];
      vehicle.lastProgressIndex = -1;
    }
  };

  ws.onclose = () => {
    pathSockets.delete(vehicleId);
  };
};

const closePathSocket = (vehicleId) => {
  const ws = pathSockets.get(vehicleId);
  if (ws) {
    ws.close();
    pathSockets.delete(vehicleId);
  }
};

const normalizeMessageTs = (ts) => {
  const n = Number(ts);
  return Number.isFinite(n) ? n : Date.now();
};

const removeVehicleFromMap = (id, options) => {
  const vehicle = options.vehiclesMap.value.get(id);
  if (!vehicle) return;
  stopVehicleAnimation(vehicle);
  vehicle.marker?.stopMove();
  vehicle.marker?.setMap(null);
  vehicle.fullPolyline?.setMap(null);
  vehicle.passedPolyline?.setMap(null);
  options.vehiclesMap.value.delete(id);
  closePathSocket(id);
};

const applyVehicleUpdate = (rawVehicle, options, meta = {}) => {
  const messageTs = normalizeMessageTs(meta.ts);
  let shouldAnimate = Boolean(meta.animate);

  const incomingCarData = normalizeVehicle(rawVehicle);
  if (!isValidLngLat(incomingCarData.currentPosition)) return;

  let vehicle = options.vehiclesMap.value.get(incomingCarData.id);
  const previousVehicleData = vehicle?.latestData;
  const carData = mergeVehicleData(previousVehicleData, incomingCarData);
  const newPosition = carData.currentPosition;

  const vehicleStore = useVehicleStore();
  vehicleStore.setVehicle(carData);

  if (!vehicle) {
    const iconUrl = options.VEHICLE_ICONS[carData.categoryId] || options.VEHICLE_ICONS.default;
    vehicle = {
      id: carData.id,
      marker: new options.AMapInstance.Marker({
        map: options.map,
        position: newPosition,
        icon: new options.AMapInstance.Icon({
          size: new options.AMapInstance.Size(40, 64),
          image: iconUrl,
          imageSize: new options.AMapInstance.Size(40, 64)
        }),
        offset: new options.AMapInstance.Pixel(-20, -32),
        angle: carData.angle || DEFAULT_STATIC_VEHICLE_ANGLE
      }),
      fullPolyline: null,
      passedPolyline: null,
      fullPathPoints: null,
      lastPosition: [...newPosition],
      lastVehicleMsgTs: 0,
      lastPathMsgTs: 0,
      activeShipmentId: null,
      activeRouteVersion: null,
      passedPathHistory: [],
      lastProgressIndex: -1,
      lastReportedPosition: [...newPosition],
      animationFrameId: null,
      segmentPauseTimer: null,
      latestData: carData
    };

    vehicle.marker.on('click', () => {
      options.imformStore.imformShow('vehicle', vehicle.latestData);
      options.map.setCenter(vehicle.marker.getPosition());
    });

    options.vehiclesMap.value.set(carData.id, vehicle);
  } else {
    vehicle.latestData = carData;
  }

  vehicle.latestData = carData;

  if (messageTs < (vehicle.lastVehicleMsgTs || 0)) {
    return;
  }
  const previousMessageTs = vehicle.lastVehicleMsgTs || 0;
  vehicle.lastVehicleMsgTs = messageTs;

  if (carData.status === 1) {
    ensurePathSocket(carData.id, options);
    vehicle.fullPolyline?.show();
  } else {
    stopVehicleAnimation(vehicle);
    vehicle.marker.stopMove();
    vehicle.fullPolyline?.hide();
    vehicle.passedPolyline?.hide();
    closePathSocket(carData.id);
    shouldAnimate = false; // Add insurance to prevent animation for non-running vehicles
  }

  const markerCurrentPosition = vehicle.marker.getPosition();
  const currentPosition = markerCurrentPosition
    ? [markerCurrentPosition.getLng(), markerCurrentPosition.getLat()]
    : vehicle.lastPosition;

  const previousReportedPosition = isValidLngLat(vehicle.lastReportedPosition)
    ? [...vehicle.lastReportedPosition]
    : null;
  vehicle.lastReportedPosition = [...newPosition];

  const distanceFromCurrent = isValidLngLat(currentPosition)
    ? options.AMapInstance.GeometryUtil.distance(currentPosition, newPosition)
    : Number.POSITIVE_INFINITY;

  if (!shouldAnimate) {
    stopVehicleAnimation(vehicle);
    vehicle.marker.stopMove();
    vehicle.marker.setPosition(newPosition);
    syncVehiclePassedPathWithPoint(vehicle, newPosition);
  } else if (!Number.isFinite(distanceFromCurrent) || distanceFromCurrent >= MAX_JUMP_DISTANCE_M) {
    stopVehicleAnimation(vehicle);
    vehicle.marker.stopMove();
    vehicle.marker.setPosition(newPosition);
    syncVehiclePassedPathWithPoint(vehicle, newPosition);
  } else if (distanceFromCurrent <= MIN_MOVE_DISTANCE_M) {
    stopVehicleAnimation(vehicle);
    vehicle.marker.stopMove();
    vehicle.marker.setPosition(newPosition);
    syncVehiclePassedPathWithPoint(vehicle, newPosition);
  } else {
    // Insurance mechanism: Always start the next segment's animation from the previous 
    // exact destination. If the animation hasn't finished, it inherently "snaps" to that corner
    // preventing diagonal arc-cutting across corners or teleportation.
    const segmentStart = isValidLngLat(previousReportedPosition)
      ? previousReportedPosition
      : currentPosition;

    const segmentDistance = isValidLngLat(segmentStart)
      ? options.AMapInstance.GeometryUtil.distance(segmentStart, newPosition)
      : Number.POSITIVE_INFINITY;

    if (!Number.isFinite(segmentDistance) || segmentDistance <= MIN_MOVE_DISTANCE_M) {
      stopVehicleAnimation(vehicle);
      vehicle.marker.stopMove();
      vehicle.marker.setPosition(newPosition);
      syncVehiclePassedPathWithPoint(vehicle, newPosition);
      vehicle.lastPosition = [...newPosition];
      vehicle.marker.setAngle(carData.angle || DEFAULT_STATIC_VEHICLE_ANGLE);
      if (options.imformStore.recentVehicle?.id === carData.id) {
        options.imformStore.imformShow('vehicle', carData);
      }
      return;
    }

    if (isValidLngLat(segmentStart)) {
      vehicle.marker.setPosition(segmentStart);
      syncVehiclePassedPathWithPoint(vehicle, segmentStart);
    }
    
    const durationMs = calcMoveDurationMs({ lastVehicleMsgTs: previousMessageTs }, messageTs, segmentDistance);
    animateMarkerSegment(vehicle, vehicle.marker, segmentStart, newPosition, durationMs);
  }

  vehicle.lastPosition = [...newPosition];

  vehicle.marker.setAngle(carData.angle || DEFAULT_STATIC_VEHICLE_ANGLE);

  if (options.imformStore.recentVehicle?.id === carData.id) {
    options.imformStore.imformShow('vehicle', carData);
  }
};

const applyVehicleSnapshot = (vehicles, options) => {
  const normalized = Array.isArray(vehicles) ? vehicles.map(normalizeVehicle) : [];

  const vehicleStore = useVehicleStore();
  vehicleStore.setVehicles(normalized);

  const incomingIds = new Set();
  normalized.forEach((v) => {
    incomingIds.add(v.id);
    applyVehicleUpdate(v, options, { animate: false, ts: Date.now() });
  });

  for (const [id] of options.vehiclesMap.value.entries()) {
    if (!incomingIds.has(id)) {
      removeVehicleFromMap(id, options);
    }
  }
};

const fetchVehicleSnapshot = async () => {
  try {
    const response = await getVehiclesData();
    if (response?.data?.code === 1 && Array.isArray(response.data.data)) {
      return response.data.data;
    }
  } catch (error) {
    console.error('车辆快照拉取失败:', error);
  }
  return [];
};

const connectVehiclesSocket = async (options) => {
  const snapshot = await fetchVehicleSnapshot();
  applyVehicleSnapshot(snapshot, options);

  vehiclesSocket = new WebSocket(`${getWsBase()}/vehicles`);

  vehiclesSocket.onmessage = (evt) => {
    let msg;
    try {
      msg = JSON.parse(evt.data);
    } catch {
      return;
    }

    const payload = msg?.payload || {};
    const event = msg?.event;

    if (event === 'snapshot') {
      applyVehicleSnapshot(payload.vehicles || [], options);
      return;
    }

    if (event === 'vehicle_move' || event === 'vehicle_update') {
      applyVehicleUpdate(payload, options, { animate: true, ts: msg?.ts });
    }
  };

  vehiclesSocket.onclose = () => {
    vehiclesSocket = null;
    if (!isRealtimeRunning) return;
    if (reconnectTimer) clearTimeout(reconnectTimer);
    reconnectTimer = setTimeout(() => {
      if (isRealtimeRunning && runtimeOptions) {
        connectVehiclesSocket(runtimeOptions);
      }
    }, 2000);
  };

  vehiclesSocket.onerror = () => {
    vehiclesSocket?.close();
  };
};

export const startPollingAndAnimation = (options) => {
  if (!options?.AMapInstance || !options?.map || isRealtimeRunning) return;
  runtimeOptions = options;
  isRealtimeRunning = true;
  options.isPollingActiveRef.value = true;
  connectVehiclesSocket(options);
};

export const pausePollingAndAnimation = (options) => {
  isRealtimeRunning = false;
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  if (vehiclesSocket) {
    vehiclesSocket.close();
    vehiclesSocket = null;
  }
  for (const vehicleId of pathSockets.keys()) {
    closePathSocket(vehicleId);
  }
  options.isPollingActiveRef.value = false;
  options.vehiclesMap.value.forEach((car) => {
    stopVehicleAnimation(car);
    car.marker?.stopMove();
    car.passedPolyline?.hide();
  });
};

export const stopPolling = () => {
  isRealtimeRunning = false;
  runtimeOptions = null;
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  if (vehiclesSocket) {
    vehiclesSocket.close();
    vehiclesSocket = null;
  }
  for (const vehicleId of pathSockets.keys()) {
    closePathSocket(vehicleId);
  }
};
