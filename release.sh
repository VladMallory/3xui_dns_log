#!/bin/bash

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Автоматический релиз${NC}"
echo "=================================="

# Функция для получения последнего тега
get_latest_tag() {
    git tag -l "v*" | sort -V | tail -n1
}

# Функция для увеличения версии
increment_version() {
    local version=$1
    if [[ $version =~ ^v([0-9]+)\.([0-9]+)$ ]]; then
        major=${BASH_REMATCH[1]}
        minor=${BASH_REMATCH[2]}
        new_minor=$((minor + 1))
        echo "v${major}.${new_minor}"
    else
        echo "v0.1"
    fi
}

# Проверяем, есть ли изменения для коммита
if [[ -z $(git status --porcelain) ]]; then
    echo -e "${YELLOW}⚠️  Нет изменений для коммита${NC}"
    read -p "Продолжить создание тега без коммита? [y/N]: " continue_without_commit
    if [[ $continue_without_commit != "y" && $continue_without_commit != "Y" ]]; then
        echo -e "${RED}❌ Релиз отменен${NC}"
        exit 1
    fi
    skip_commit=true
else
    skip_commit=false
fi

# Получаем текущий максимальный тег
current_tag=$(get_latest_tag)
if [[ -z $current_tag ]]; then
    new_tag="v0.1"
    echo -e "${YELLOW}📋 Текущих тегов не найдено, начинаем с $new_tag${NC}"
else
    new_tag=$(increment_version $current_tag)
    echo -e "${GREEN}📋 Текущий тег: $current_tag${NC}"
fi

echo -e "${BLUE}🏷️  Новый тег: $new_tag${NC}"

# Запрашиваем сообщение коммита и тега
if [[ $skip_commit == false ]]; then
    echo ""
    read -p "💬 Введите сообщение коммита: " commit_message
    if [[ -z "$commit_message" ]]; then
        commit_message="Release $new_tag"
    fi
fi

echo ""
read -p "🏷️  Введите сообщение для тега [$new_tag]: " tag_message
if [[ -z "$tag_message" ]]; then
    tag_message="Release $new_tag"
fi

# Подтверждение
echo ""
echo -e "${YELLOW}📝 Сводка релиза:${NC}"
echo "  Новый тег: $new_tag"
if [[ $skip_commit == false ]]; then
    echo "  Сообщение коммита: $commit_message"
fi
echo "  Сообщение тега: $tag_message"
echo ""

read -p "🚀 Продолжить релиз? [Y/n]: " confirm
if [[ $confirm == "n" || $confirm == "N" ]]; then
    echo -e "${RED}❌ Релиз отменен${NC}"
    exit 1
fi

echo ""
echo -e "${BLUE}🔄 Выполняем релиз...${NC}"

# Выполняем git команды
if [[ $skip_commit == false ]]; then
    echo -e "${YELLOW}📦 Добавляем файлы...${NC}"
    git add .
    
    echo -e "${YELLOW}💾 Создаем коммит...${NC}"
    git commit -m "$commit_message"
    
    if [[ $? -ne 0 ]]; then
        echo -e "${RED}❌ Ошибка при создании коммита${NC}"
        exit 1
    fi
fi

echo -e "${YELLOW}🏷️  Создаем тег...${NC}"
git tag -a "$new_tag" -m "$tag_message"

if [[ $? -ne 0 ]]; then
    echo -e "${RED}❌ Ошибка при создании тега${NC}"
    exit 1
fi

echo -e "${YELLOW}📤 Отправляем тег...${NC}"
git push origin "$new_tag"

if [[ $? -ne 0 ]]; then
    echo -e "${RED}❌ Ошибка при отправке тега${NC}"
    exit 1
fi

# Проверяем, нужно ли пушить в main
current_branch=$(git branch --show-current)
if [[ $current_branch != "main" ]]; then
    read -p "📤 Отправить изменения в ветку main? [y/N]: " push_main
    if [[ $push_main == "y" || $push_main == "Y" ]]; then
        echo -e "${YELLOW}📤 Отправляем в main...${NC}"
        git push origin main
        
        if [[ $? -ne 0 ]]; then
            echo -e "${RED}❌ Ошибка при отправке в main${NC}"
        fi
    fi
else
    if [[ $skip_commit == false ]]; then
        echo -e "${YELLOW}📤 Отправляем в main...${NC}"
        git push origin main
        
        if [[ $? -ne 0 ]]; then
            echo -e "${RED}❌ Ошибка при отправке в main${NC}"
        fi
    fi
fi

echo ""
echo -e "${GREEN}✅ Релиз $new_tag успешно создан!${NC}"
echo -e "${GREEN}🎉 Все команды выполнены успешно${NC}"

