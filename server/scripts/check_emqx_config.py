#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
EMQX配置检查脚本
检查EMQX的认证器和授权器配置
"""

import requests
import json

# EMQX配置
EMQX_CONFIG = {
    "host": "192.168.50.194",
    "dashboard_port": 18083,
    "username": "admin",
    "password": "xuehua123"
}

def get_api_token():
    """获取API Token"""
    url = f"http://{EMQX_CONFIG['host']}:{EMQX_CONFIG['dashboard_port']}/api/v5/login"
    
    response = requests.post(url, json={
        "username": EMQX_CONFIG["username"],
        "password": EMQX_CONFIG["password"]
    })
    
    if response.status_code == 200:
        token = response.json().get("token")
        return token
    else:
        print(f"❌ 获取API Token失败: {response.status_code}")
        return None

def check_authenticators(token):
    """检查认证器配置"""
    print("🔍 检查认证器配置...")
    
    url = f"http://{EMQX_CONFIG['host']}:{EMQX_CONFIG['dashboard_port']}/api/v5/authentication"
    headers = {"Authorization": f"Bearer {token}"}
    
    response = requests.get(url, headers=headers)
    if response.status_code == 200:
        auth_config = response.json()
        print("✅ 认证器配置获取成功")
        print(json.dumps(auth_config, indent=2, ensure_ascii=False))
        return auth_config
    else:
        print(f"❌ 获取认证器配置失败: {response.status_code}")
        print(response.text)
        return None

def check_authorization(token):
    """检查授权器配置"""
    print("\n🔍 检查授权器配置...")
    
    url = f"http://{EMQX_CONFIG['host']}:{EMQX_CONFIG['dashboard_port']}/api/v5/authorization/settings"
    headers = {"Authorization": f"Bearer {token}"}
    
    response = requests.get(url, headers=headers)
    if response.status_code == 200:
        authz_config = response.json()
        print("✅ 授权器配置获取成功")
        print(json.dumps(authz_config, indent=2, ensure_ascii=False))
        return authz_config
    else:
        print(f"❌ 获取授权器配置失败: {response.status_code}")
        print(response.text)
        return None

def check_authorization_sources(token):
    """检查授权源配置"""
    print("\n🔍 检查授权源配置...")
    
    url = f"http://{EMQX_CONFIG['host']}:{EMQX_CONFIG['dashboard_port']}/api/v5/authorization/sources"
    headers = {"Authorization": f"Bearer {token}"}
    
    response = requests.get(url, headers=headers)
    if response.status_code == 200:
        authz_sources = response.json()
        print("✅ 授权源配置获取成功")
        print(json.dumps(authz_sources, indent=2, ensure_ascii=False))
        return authz_sources
    else:
        print(f"❌ 获取授权源配置失败: {response.status_code}")
        print(response.text)
        return None

def check_webhooks(token):
    """检查Webhook配置"""
    print("\n🔍 检查Webhook配置...")
    
    # 检查Actions (Webhooks)
    url = f"http://{EMQX_CONFIG['host']}:{EMQX_CONFIG['dashboard_port']}/api/v5/actions"
    headers = {"Authorization": f"Bearer {token}"}
    
    response = requests.get(url, headers=headers)
    if response.status_code == 200:
        webhooks = response.json()
        print("✅ Webhook配置获取成功")
        print(json.dumps(webhooks, indent=2, ensure_ascii=False))
        return webhooks
    else:
        print(f"❌ 获取Webhook配置失败: {response.status_code}")
        print(response.text)
        return None

def main():
    print("="*60)
    print("🚀 开始检查EMQX配置")
    print("="*60)
    
    # 获取API Token
    token = get_api_token()
    if not token:
        return
    
    print(f"✅ API Token获取成功")
    
    # 检查各项配置
    auth_config = check_authenticators(token)
    authz_config = check_authorization(token)
    authz_sources = check_authorization_sources(token)
    webhook_config = check_webhooks(token)
    
    print("\n" + "="*60)
    print("📊 配置检查总结")
    print("="*60)
    
    if auth_config:
        print("✅ 认证器配置可访问")
        if auth_config and len(auth_config) > 0:
            for i, auth in enumerate(auth_config):
                mechanism = auth.get('mechanism', 'unknown')
                backend = auth.get('backend', 'unknown')
                enable = auth.get('enable', False)
                print(f"   认证器 {i+1}: {mechanism}/{backend} (启用: {enable})")
        else:
            print("   ⚠️  未发现已配置的认证器")
    
    if authz_config:
        print("✅ 授权器配置可访问")
    
    if authz_sources:
        print("✅ 授权源配置可访问")
        sources = authz_sources.get('sources', []) if isinstance(authz_sources, dict) else authz_sources
        if sources and len(sources) > 0:
            for i, source in enumerate(sources):
                if isinstance(source, dict):
                    source_type = source.get('type', 'unknown')
                    enable = source.get('enable', False)
                    url = source.get('url', 'N/A')
                    print(f"   授权源 {i+1}: {source_type} (启用: {enable}) URL: {url}")
                else:
                    print(f"   授权源 {i+1}: {source}")
        else:
            print("   ⚠️  未发现已配置的授权源")
    
    if webhook_config:
        print("✅ Webhook配置可访问")
        if webhook_config and len(webhook_config) > 0:
            for i, webhook in enumerate(webhook_config):
                name = webhook.get('name', 'unknown')
                type_name = webhook.get('type', 'unknown')
                enable = webhook.get('enable', False)
                print(f"   Webhook {i+1}: {name} ({type_name}) (启用: {enable})")
        else:
            print("   ⚠️  未发现已配置的Webhook")

if __name__ == "__main__":
    main() 