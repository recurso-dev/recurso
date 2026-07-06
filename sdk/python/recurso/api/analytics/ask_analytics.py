from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.ask_analytics_body import AskAnalyticsBody
from ...models.ask_analytics_response_200 import AskAnalyticsResponse200
from ...models.error import Error
from ...types import Response


def _get_kwargs(
    *,
    body: AskAnalyticsBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/analytics/ask",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> AskAnalyticsResponse200 | Error | None:
    if response.status_code == 200:
        response_200 = AskAnalyticsResponse200.from_dict(response.json())

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
) -> Response[AskAnalyticsResponse200 | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: AskAnalyticsBody,
) -> Response[AskAnalyticsResponse200 | Error]:
    """Ask a natural-language analytics question

     Uses an LLM to translate the question into tenant-scoped SQL, runs it, and returns both the rows and
    the generated query.

    Args:
        body (AskAnalyticsBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AskAnalyticsResponse200 | Error]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    body: AskAnalyticsBody,
) -> AskAnalyticsResponse200 | Error | None:
    """Ask a natural-language analytics question

     Uses an LLM to translate the question into tenant-scoped SQL, runs it, and returns both the rows and
    the generated query.

    Args:
        body (AskAnalyticsBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AskAnalyticsResponse200 | Error
    """

    return sync_detailed(
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: AskAnalyticsBody,
) -> Response[AskAnalyticsResponse200 | Error]:
    """Ask a natural-language analytics question

     Uses an LLM to translate the question into tenant-scoped SQL, runs it, and returns both the rows and
    the generated query.

    Args:
        body (AskAnalyticsBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AskAnalyticsResponse200 | Error]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: AskAnalyticsBody,
) -> AskAnalyticsResponse200 | Error | None:
    """Ask a natural-language analytics question

     Uses an LLM to translate the question into tenant-scoped SQL, runs it, and returns both the rows and
    the generated query.

    Args:
        body (AskAnalyticsBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AskAnalyticsResponse200 | Error
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
        )
    ).parsed
