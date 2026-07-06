import datetime
from http import HTTPStatus
from typing import Any
from uuid import UUID

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.query_usage_granularity import QueryUsageGranularity
from ...models.query_usage_response_200 import QueryUsageResponse200
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    subscription_id: UUID | Unset = UNSET,
    customer_id: UUID | Unset = UNSET,
    dimension: str | Unset = UNSET,
    from_: datetime.datetime | Unset = UNSET,
    to: datetime.datetime | Unset = UNSET,
    granularity: QueryUsageGranularity | Unset = QueryUsageGranularity.DAY,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    json_subscription_id: str | Unset = UNSET
    if not isinstance(subscription_id, Unset):
        json_subscription_id = str(subscription_id)
    params["subscription_id"] = json_subscription_id

    json_customer_id: str | Unset = UNSET
    if not isinstance(customer_id, Unset):
        json_customer_id = str(customer_id)
    params["customer_id"] = json_customer_id

    params["dimension"] = dimension

    json_from_: str | Unset = UNSET
    if not isinstance(from_, Unset):
        json_from_ = from_.isoformat()
    params["from"] = json_from_

    json_to: str | Unset = UNSET
    if not isinstance(to, Unset):
        json_to = to.isoformat()
    params["to"] = json_to

    json_granularity: str | Unset = UNSET
    if not isinstance(granularity, Unset):
        json_granularity = granularity.value

    params["granularity"] = json_granularity

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/usage",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | QueryUsageResponse200 | None:
    if response.status_code == 200:
        response_200 = QueryUsageResponse200.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())

        return response_400

    if response.status_code == 401:
        response_401 = Error.from_dict(response.json())

        return response_401

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Error | QueryUsageResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    subscription_id: UUID | Unset = UNSET,
    customer_id: UUID | Unset = UNSET,
    dimension: str | Unset = UNSET,
    from_: datetime.datetime | Unset = UNSET,
    to: datetime.datetime | Unset = UNSET,
    granularity: QueryUsageGranularity | Unset = QueryUsageGranularity.DAY,
) -> Response[Error | QueryUsageResponse200]:
    """Time-windowed usage buckets

     Aggregates usage events into `date_trunc`'d time buckets. At least one of `subscription_id` or
    `customer_id` is required. The window defaults to the last 30 days.

    Args:
        subscription_id (UUID | Unset):
        customer_id (UUID | Unset):
        dimension (str | Unset):
        from_ (datetime.datetime | Unset):
        to (datetime.datetime | Unset):
        granularity (QueryUsageGranularity | Unset):  Default: QueryUsageGranularity.DAY.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | QueryUsageResponse200]
    """

    kwargs = _get_kwargs(
        subscription_id=subscription_id,
        customer_id=customer_id,
        dimension=dimension,
        from_=from_,
        to=to,
        granularity=granularity,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    subscription_id: UUID | Unset = UNSET,
    customer_id: UUID | Unset = UNSET,
    dimension: str | Unset = UNSET,
    from_: datetime.datetime | Unset = UNSET,
    to: datetime.datetime | Unset = UNSET,
    granularity: QueryUsageGranularity | Unset = QueryUsageGranularity.DAY,
) -> Error | QueryUsageResponse200 | None:
    """Time-windowed usage buckets

     Aggregates usage events into `date_trunc`'d time buckets. At least one of `subscription_id` or
    `customer_id` is required. The window defaults to the last 30 days.

    Args:
        subscription_id (UUID | Unset):
        customer_id (UUID | Unset):
        dimension (str | Unset):
        from_ (datetime.datetime | Unset):
        to (datetime.datetime | Unset):
        granularity (QueryUsageGranularity | Unset):  Default: QueryUsageGranularity.DAY.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | QueryUsageResponse200
    """

    return sync_detailed(
        client=client,
        subscription_id=subscription_id,
        customer_id=customer_id,
        dimension=dimension,
        from_=from_,
        to=to,
        granularity=granularity,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    subscription_id: UUID | Unset = UNSET,
    customer_id: UUID | Unset = UNSET,
    dimension: str | Unset = UNSET,
    from_: datetime.datetime | Unset = UNSET,
    to: datetime.datetime | Unset = UNSET,
    granularity: QueryUsageGranularity | Unset = QueryUsageGranularity.DAY,
) -> Response[Error | QueryUsageResponse200]:
    """Time-windowed usage buckets

     Aggregates usage events into `date_trunc`'d time buckets. At least one of `subscription_id` or
    `customer_id` is required. The window defaults to the last 30 days.

    Args:
        subscription_id (UUID | Unset):
        customer_id (UUID | Unset):
        dimension (str | Unset):
        from_ (datetime.datetime | Unset):
        to (datetime.datetime | Unset):
        granularity (QueryUsageGranularity | Unset):  Default: QueryUsageGranularity.DAY.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | QueryUsageResponse200]
    """

    kwargs = _get_kwargs(
        subscription_id=subscription_id,
        customer_id=customer_id,
        dimension=dimension,
        from_=from_,
        to=to,
        granularity=granularity,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    subscription_id: UUID | Unset = UNSET,
    customer_id: UUID | Unset = UNSET,
    dimension: str | Unset = UNSET,
    from_: datetime.datetime | Unset = UNSET,
    to: datetime.datetime | Unset = UNSET,
    granularity: QueryUsageGranularity | Unset = QueryUsageGranularity.DAY,
) -> Error | QueryUsageResponse200 | None:
    """Time-windowed usage buckets

     Aggregates usage events into `date_trunc`'d time buckets. At least one of `subscription_id` or
    `customer_id` is required. The window defaults to the last 30 days.

    Args:
        subscription_id (UUID | Unset):
        customer_id (UUID | Unset):
        dimension (str | Unset):
        from_ (datetime.datetime | Unset):
        to (datetime.datetime | Unset):
        granularity (QueryUsageGranularity | Unset):  Default: QueryUsageGranularity.DAY.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | QueryUsageResponse200
    """

    return (
        await asyncio_detailed(
            client=client,
            subscription_id=subscription_id,
            customer_id=customer_id,
            dimension=dimension,
            from_=from_,
            to=to,
            granularity=granularity,
        )
    ).parsed
