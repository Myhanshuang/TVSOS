package com.tvsos.config;

import com.fasterxml.jackson.annotation.JsonTypeInfo;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.fasterxml.jackson.databind.jsontype.impl.LaissezFaireSubTypeValidator;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import lombok.extern.slf4j.Slf4j;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.redis.connection.RedisConnectionFactory;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.serializer.GenericJackson2JsonRedisSerializer;
import org.springframework.data.redis.serializer.StringRedisSerializer;

@Slf4j
@Configuration
public class RedisTemplateConfig {
    @Bean
    public RedisTemplate<String, Object> redisTemplate(RedisConnectionFactory factory){
        log.info("开始创建redis模版对象...");
        RedisTemplate<String, Object> redisTemplate = new RedisTemplate<>();
        //设置连接工厂
        redisTemplate.setConnectionFactory(factory);

        ObjectMapper objectMapper = new ObjectMapper();
        // 设置支持LocalDateTime
        objectMapper.registerModule(new JavaTimeModule()); // 支持Java 8时间类型
        objectMapper.disable(SerializationFeature.WRITE_DATES_AS_TIMESTAMPS); // 时间序列化为字符串

        // 开启类型信息，解决反序列化时对象类型丢失的问题
        objectMapper.activateDefaultTyping(
                LaissezFaireSubTypeValidator.instance,
                ObjectMapper.DefaultTyping.NON_FINAL,
                JsonTypeInfo.As.PROPERTY
        );

        //设置key的序列化方式：String
        redisTemplate.setKeySerializer(new StringRedisSerializer());

        //设置hash/key的序列化方式：String
        redisTemplate.setHashKeySerializer(new StringRedisSerializer());

        //设置value的序列化方式：json
        GenericJackson2JsonRedisSerializer jsonSerializer = new GenericJackson2JsonRedisSerializer(objectMapper);
        redisTemplate.setValueSerializer(jsonSerializer);
        redisTemplate.setHashValueSerializer(jsonSerializer);

        //初始化redis模版对象
        redisTemplate.afterPropertiesSet();
        return redisTemplate;
    }
}
