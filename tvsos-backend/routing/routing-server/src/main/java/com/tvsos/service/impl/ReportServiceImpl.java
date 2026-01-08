package com.tvsos.service.impl;

import com.tvsos.service.*;
import entity.Poi;
import entity.Vehicle;
import entity.VehicleCategory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import vo.VehicleCategoryVO;
import vo.VehicleVO;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

@Service
public class ReportServiceImpl implements ReportService {
    @Autowired
    private VehicleCategoryService vehicleCategoryService;
    @Autowired
    private VehicleService vehicleService;
    @Autowired
    private PoiService poiService;
    @Autowired
    private DriverService driverService;

    /**
     * 饼图 车辆类型 - 数量
     * @return
     */
    @Override
    public List<VehicleCategoryVO> reportVehicleCategory() {
        Map<Integer, Integer> hashMap = new HashMap<>();
        // 获取所有车辆 category
        List<VehicleCategory> vehicleCategoryList = vehicleCategoryService.findAll();
        for(VehicleCategory vehicleCategory : vehicleCategoryList) {
            // 每个车辆 category 的 id
            Long categoryId = vehicleCategory.getId();
            // 查找每个 categoryId 有多少辆车
            Integer cnt = vehicleService.countVehicleCategory(categoryId);
            // 当前 category 的种类
            Integer tybe = vehicleCategory.getTybe();
            Integer i = hashMap.get(tybe);
            if(i == null){
                i = 0;
            }
            hashMap.put(tybe, i + cnt);
        }
        List<VehicleCategoryVO> resList = new ArrayList<>();
        for(int i = 1; i <= 5; i++){
            VehicleCategoryVO vo = new VehicleCategoryVO();
            String name = null;
            switch (i){
                case 1: name = "平板"; break;
                case 2: name = "高护栏"; break;
                case 3: name = "厢式"; break;
                case 4: name = "冷链"; break;
                case 5: name = "危化品"; break;
            }
            vo.setName(name);
            Integer value = hashMap.get(i);
            if(value == null){
                value = 0;
            }
            vo.setCount(value);
            resList.add(vo);
        }
        return resList;
    }

    /**
     * 柱状图 poi类型 - 数量
     * @return
     */
    @Override
    public List<List> reportPoiTybe() {
        // 获取所有poi
        List<Poi> poiList = poiService.findAll();
        /**
         *      "加油站": 1,
         *     "加气站": 2,
         *     "其它能源站": 3,
         *     "工厂": 4,
         *     "购物相关场所": 5,
         *     "家居建材市场": 6,
         *     "公司企业": 7,
         */
        List<String> strList = new ArrayList<>();
        List<Integer> countList = new ArrayList<>();
        Integer sum = 0;
        strList.add("加油站");
        strList.add("加气站");
        strList.add("其他能源站");
        strList.add("工厂");
        strList.add("购物相关场所");
        strList.add("家居建材市场");
        strList.add("公司企业");
        strList.add("其他"); // 8
        for(int i = 1; i <= 7; i++) {
            Integer tybe = i;
            Integer count = poiService.countTybe(tybe);
            sum += count;
            countList.add(count);
        }
        countList.add(poiList.size() - sum);
        List<List> resList = new ArrayList<>();
        resList.add(strList);
        resList.add(countList);
        return resList;
    }

    /**
     * 统计 车上的货物总量
     * @return
     */
    @Override
    public Double reportCargoSize() {
        Double sum = vehicleService.sumCargoSize();
        return sum;
    }

    /**
     * 统计 poi 数量
     * @return
     */
    @Override
    public Integer reportPoiSum() {
        Integer sum = poiService.count();
        return sum;
    }

    /**
     * 统计车辆数量
     * @return
     */
    @Override
    public Integer reportVehicleSum() {
        Integer sum =  vehicleService.count();
        return sum;
    }

    /**
     * 统计司机数量
     * @return
     */
    @Override
    public Integer reportDriverSum() {
        Integer sum = driverService.count();
        return sum;
    }
}
