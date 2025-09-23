package com.tvsos.utils;

import com.alibaba.fastjson2.JSONObject;
import org.apache.hc.client5.http.classic.methods.HttpGet;
import org.apache.hc.client5.http.classic.methods.HttpPost;
import org.apache.hc.client5.http.config.RequestConfig;
import org.apache.hc.client5.http.impl.classic.CloseableHttpClient;
import org.apache.hc.client5.http.impl.classic.CloseableHttpResponse;
import org.apache.hc.client5.http.impl.classic.HttpClients;
import org.apache.hc.core5.http.ContentType;
import org.apache.hc.core5.http.ParseException;
import org.apache.hc.core5.http.io.entity.EntityUtils;
import org.apache.hc.core5.http.io.entity.StringEntity;
import org.apache.hc.core5.net.URIBuilder;
import org.apache.hc.core5.util.Timeout;

import java.io.IOException;
import java.net.URI;
import java.nio.charset.StandardCharsets;
import java.util.Map;

/**
 * Http工具类（HttpClient5 + fastjson2）
 * 统一采用 JSON 作为请求体
 */
public class HttpUtils {

    // 超时时间：5秒
    private static final int TIMEOUT_MSEC = 5 * 1000;

    /**
     * GET 请求
     */
    public static String doGet(String url, Map<String, String> paramMap) {
        try (CloseableHttpClient httpClient = HttpClients.createDefault()) {
            URIBuilder builder = new URIBuilder(url);
            if (paramMap != null) {
                for (Map.Entry<String, String> entry : paramMap.entrySet()) {
                    builder.addParameter(entry.getKey(), entry.getValue());
                }
            }
            URI uri = builder.build();
            HttpGet httpGet = new HttpGet(uri);
            httpGet.setConfig(builderRequestConfig());

            try (CloseableHttpResponse response = httpClient.execute(httpGet)) {
                if (response.getCode() == 200) {
                    return EntityUtils.toString(response.getEntity(), StandardCharsets.UTF_8);
                }
            }
        } catch (Exception e) {
            e.printStackTrace();
        }
        return "";
    }

    /**
     * POST 请求（JSON 格式）
     */
    public static String doPost(String url, Map<String, Object> paramMap) {
        try (CloseableHttpClient httpClient = HttpClients.createDefault()) {
            HttpPost httpPost = new HttpPost(url);

            if (paramMap != null) {
                JSONObject jsonObject = new JSONObject(paramMap);
                StringEntity entity = new StringEntity(
                        jsonObject.toString(),
                        ContentType.APPLICATION_JSON.withCharset(StandardCharsets.UTF_8)
                );
                httpPost.setEntity(entity);
            }

            httpPost.setConfig(builderRequestConfig());

            try (CloseableHttpResponse response = httpClient.execute(httpPost)) {
                return EntityUtils.toString(response.getEntity(), StandardCharsets.UTF_8);
            } catch (ParseException e) {
                throw new RuntimeException("解析响应失败", e);
            }
        } catch (IOException e) {
            throw new RuntimeException("请求失败", e);
        }
    }

    /**
     * 构建请求配置
     */
    private static RequestConfig builderRequestConfig() {
        return RequestConfig.custom()
                .setConnectTimeout(Timeout.ofMilliseconds(TIMEOUT_MSEC))  // 连接超时
                .setResponseTimeout(Timeout.ofMilliseconds(TIMEOUT_MSEC)) // 响应超时
                .build();
    }
}
